package biz

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	v1 "github.com/xuanyiying/smart-park/api/payment/v1"

	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

const (
	SecurityEventAmountMismatch    = "amount_mismatch"
	SecurityEventInvalidStatus     = "invalid_status"
	SecurityEventDuplicateCallback = "duplicate_callback"
	SecurityEventSignatureFailed   = "signature_failed"
	SecurityEventInvalidTradeNo    = "invalid_trade_no"
	SecurityEventOrderNotFound     = "order_not_found"
	SecurityEventNotConfigured     = "gateway_not_configured"
	SecurityEventUnknown           = "unknown"
)

// Callback processing errors. They are intentionally opaque: the response returned to the
// gateway never explains why a callback was rejected, to avoid leaking order internals.
var (
	ErrGatewayNotConfigured = errors.New("payment gateway is not configured")
	ErrRawPayloadMissing    = errors.New("callback received without its raw HTTP payload")
)

// RawCallback carries the untouched HTTP payload of a payment gateway callback.
//
// Gateway signatures are computed over the exact bytes that were transmitted. Once the
// request has been decoded into a protobuf message the original field order, encoding and
// unknown parameters are all lost, so verification must run against this raw payload.
// Anything else is unverifiable and must not be trusted.
type RawCallback struct {
	Headers map[string]string
	Body    []byte
	Form    url.Values
}

type SecurityEvent struct {
	Type        string
	OrderID     string
	Expected    float64
	Received    float64
	Transaction string
	Timestamp   time.Time
	Details     string
}

type GateControlService interface {
	OpenGate(ctx context.Context, deviceID string, recordID string) error
}

type RecordRepo interface {
	GetRecord(ctx context.Context, recordID string) (*ParkingRecordInfo, error)
	UpdateRecordStatus(ctx context.Context, recordID string, status string) error
}

type ParkingRecordInfo struct {
	ID           string
	ExitDeviceID string
	LotID        string
	PlateNumber  string
	// FinalAmount is the server-side fee (in cents) calculated at exit time.
	// Zero means the record carries no authoritative amount yet.
	FinalAmount int64
}

// settlementOutcome summarises how a verified callback was applied.
type settlementOutcome struct {
	// Duplicate is true when the order had already been settled. The gateway is told the
	// callback succeeded so that it stops retrying.
	Duplicate bool
}

// HandleWechatCallback is the gRPC entrypoint for WeChat notifications.
//
// It intentionally refuses to process anything. A protobuf message carries no signature
// material, so accepting one would let any caller mark an order as paid. Real WeChat
// callbacks arrive over HTTP and are handled by HandleWechatNotification, which verifies
// the APIv3 signature over the raw request body.
func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
	uc.logSecurityEvent(ctx, SecurityEventSignatureFailed, req.OutTradeNo, 0, parseAmount(req.TotalFee), req.TransactionId,
		"rejected unsigned callback: WeChat notifications must be verified over the raw HTTP payload")
	return &v1.WechatCallbackResponse{
		ReturnCode: string(WechatStatusFail),
		ReturnMsg:  "unsigned callback rejected",
	}, nil
}

// HandleWechatNotification verifies and applies a WeChat Pay APIv3 callback.
//
// Pipeline: configuration check -> timestamp freshness -> RSA-PSS signature over the raw
// body -> AES-256-GCM decryption -> amount and status checks -> conditional (idempotent)
// database update -> automatic gate opening.
func (uc *PaymentUseCase) HandleWechatNotification(ctx context.Context, raw *RawCallback) (*v1.WechatCallbackResponse, error) {
	if raw == nil || len(raw.Body) == 0 {
		return uc.buildWechatErrorResponse("missing notification payload"), nil
	}
	if !uc.config.WechatPayEnabled() {
		uc.logSecurityEvent(ctx, SecurityEventNotConfigured, "", 0, 0, "", "wechat pay callbacks are not configured")
		return uc.buildWechatErrorResponse("gateway not configured"), nil
	}
	if uc.wechatClient == nil {
		uc.logSecurityEvent(ctx, SecurityEventNotConfigured, "", 0, 0, "", "wechat client unavailable")
		return uc.buildWechatErrorResponse("gateway unavailable"), nil
	}

	transaction, err := uc.wechatClient.ParseAndVerifyCallback(raw.Headers, raw.Body, time.Now())
	if err != nil {
		uc.logSecurityEvent(ctx, SecurityEventSignatureFailed, "", 0, 0, "", err.Error())
		return uc.buildWechatErrorResponse("signature verification failed"), nil
	}

	if transaction.TradeState != wechat.TradeStateSuccess {
		uc.log.WithContext(ctx).Infof("wechat callback trade state %q is not a successful payment, ignoring", transaction.TradeState)
		// Not an error: WeChat also notifies about closed/refunded transactions.
		return &v1.WechatCallbackResponse{
			ReturnCode: string(WechatStatusSuccess),
			ReturnMsg:  "OK",
		}, nil
	}

	orderID, err := uuid.Parse(transaction.OutTradeNo)
	if err != nil {
		uc.logSecurityEvent(ctx, SecurityEventInvalidTradeNo, transaction.OutTradeNo, 0,
			float64(transaction.Amount.Total)/100, transaction.TransactionID, err.Error())
		return uc.buildWechatErrorResponse("invalid out_trade_no"), nil
	}

	outcome, err := uc.settleOrder(ctx, orderID, MethodWechat, transaction.TransactionID, transaction.Amount.Total)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to settle order %s: %v", orderID, err)
		return uc.buildWechatErrorResponse("settlement failed"), nil
	}
	if outcome.Duplicate {
		uc.logSecurityEvent(ctx, SecurityEventDuplicateCallback, orderID.String(), 0, 0, transaction.TransactionID,
			"order was already settled, callback acknowledged without changes")
	}

	return &v1.WechatCallbackResponse{
		ReturnCode: string(WechatStatusSuccess),
		ReturnMsg:  "OK",
	}, nil
}

// HandleAlipayCallback is the gRPC entrypoint for Alipay notifications and, like its
// WeChat counterpart, refuses to trust an unverifiable payload.
func (uc *PaymentUseCase) HandleAlipayCallback(ctx context.Context, req *v1.AlipayCallbackRequest) (*v1.AlipayCallbackResponse, error) {
	uc.logSecurityEvent(ctx, SecurityEventSignatureFailed, req.OutTradeNo, 0, parseAmountFloat(req.TotalAmount), req.TradeNo,
		"rejected unsigned callback: Alipay notifications must be verified over the raw HTTP form")
	return &v1.AlipayCallbackResponse{
		Code: "FAIL",
		Msg:  "unsigned callback rejected",
	}, nil
}

// HandleAlipayNotification verifies and applies an Alipay callback.
//
// Alipay signs every received parameter (excluding sign/sign_type) sorted by key, so the
// verification runs over the complete urlencoded form rather than a hand-picked subset.
func (uc *PaymentUseCase) HandleAlipayNotification(ctx context.Context, raw *RawCallback) (*v1.AlipayCallbackResponse, error) {
	if raw == nil || len(raw.Form) == 0 {
		return uc.buildAlipayErrorResponse("missing notification payload"), nil
	}
	if !uc.config.AlipayEnabled() || uc.alipayClient == nil {
		uc.logSecurityEvent(ctx, SecurityEventNotConfigured, "", 0, 0, "", "alipay callbacks are not configured")
		return uc.buildAlipayErrorResponse("gateway not configured"), nil
	}

	if err := uc.alipayClient.VerifyNotification(raw.Form); err != nil {
		uc.logSecurityEvent(ctx, SecurityEventSignatureFailed, raw.Form.Get("out_trade_no"), 0,
			parseAmountFloat(raw.Form.Get("total_amount")), raw.Form.Get("trade_no"), err.Error())
		return uc.buildAlipayErrorResponse("signature verification failed"), nil
	}

	tradeStatus := raw.Form.Get("trade_status")
	if !isAlipaySuccessStatus(tradeStatus) {
		uc.log.WithContext(ctx).Infof("alipay callback trade status %q is not a successful payment, ignoring", tradeStatus)
		return &v1.AlipayCallbackResponse{Code: "success", Msg: "OK"}, nil
	}

	outTradeNo := raw.Form.Get("out_trade_no")
	orderID, err := uuid.Parse(outTradeNo)
	if err != nil {
		uc.logSecurityEvent(ctx, SecurityEventInvalidTradeNo, outTradeNo, 0,
			parseAmountFloat(raw.Form.Get("total_amount")), raw.Form.Get("trade_no"), err.Error())
		return uc.buildAlipayErrorResponse("invalid out_trade_no"), nil
	}

	// Alipay reports amounts in yuan with two decimals; convert to cents so that the
	// comparison against the order is done in integers.
	paidCents, err := yuanStringToCents(raw.Form.Get("total_amount"))
	if err != nil {
		uc.logSecurityEvent(ctx, SecurityEventAmountMismatch, outTradeNo, 0, 0, raw.Form.Get("trade_no"), err.Error())
		return uc.buildAlipayErrorResponse("invalid total_amount"), nil
	}

	outcome, err := uc.settleOrder(ctx, orderID, MethodAlipay, raw.Form.Get("trade_no"), paidCents)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to settle order %s: %v", orderID, err)
		return uc.buildAlipayErrorResponse("settlement failed"), nil
	}
	if outcome.Duplicate {
		uc.logSecurityEvent(ctx, SecurityEventDuplicateCallback, orderID.String(), 0, 0, raw.Form.Get("trade_no"),
			"order was already settled, callback acknowledged without changes")
	}

	return &v1.AlipayCallbackResponse{Code: "success", Msg: "OK"}, nil
}

// settleOrder applies a verified payment to an order.
//
// Idempotency is delegated to the database: MarkOrderPaid only flips rows that are still
// pending, so duplicate callbacks, concurrent replicas and replays after a restart all
// converge on exactly one settlement.
func (uc *PaymentUseCase) settleOrder(ctx context.Context, orderID uuid.UUID, method PayMethod, transactionID string, paidCents int64) (*settlementOutcome, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("gateway did not provide a transaction id for order %s", orderID)
	}

	order, err := uc.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if order == nil {
		uc.logSecurityEvent(ctx, SecurityEventOrderNotFound, orderID.String(), 0, float64(paidCents)/100, transactionID,
			"no order matches the callback out_trade_no")
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// Both sides are cents (分) now; previously the order held 元 and this
	// comparison was a float-equal with an explicit 0.01 tolerance.
	if order.FinalAmount != paidCents {
		uc.logSecurityEvent(ctx, SecurityEventAmountMismatch, orderID.String(), float64(order.FinalAmount)/100,
			float64(paidCents)/100, transactionID, "gateway amount does not match the order amount")
		return nil, fmt.Errorf("amount mismatch: order %s expected %d cents, gateway reported %d cents",
			orderID, order.FinalAmount, paidCents)
	}

	if order.Status != string(StatusPending) {
		// Already paid, refunded or failed. The gateway must be told to stop retrying, but
		// we record the event so that an unexpected sequence can be investigated.
		uc.logSecurityEvent(ctx, SecurityEventInvalidStatus, orderID.String(), float64(order.FinalAmount)/100,
			float64(paidCents)/100, transactionID, fmt.Sprintf("status: %s", order.Status))
		return &settlementOutcome{Duplicate: true}, nil
	}

	// The order update and the order.settled event must commit together: the
	// event is what later opens the gate, so an update without its event would
	// strand the driver at the barrier. MarkOrderPaid is a conditional update,
	// which keeps this whole block idempotent across transaction retries.
	duplicate := false
	err = uc.orderRepo.WithTx(ctx, func(ctx context.Context) error {
		applied, err := uc.orderRepo.MarkOrderPaid(ctx, orderID, string(method), transactionID, order.FinalAmount, time.Now())
		if err != nil {
			return fmt.Errorf("failed to mark order paid: %w", err)
		}
		if !applied {
			// Lost the race against another replica or a retried callback.
			duplicate = true
			return nil
		}

		if uc.outbox != nil {
			return uc.publishOrderSettled(ctx, order, method, transactionID)
		}

		// No outbox attached (tests, minimal deployments): run the side effect
		// inline as before.
		order.Status = string(StatusPaid)
		order.PayMethod = string(method)
		order.TransactionID = transactionID
		order.PaidAmount = order.FinalAmount
		if err := uc.applyOrderSettledSideEffects(ctx, order); err != nil {
			uc.log.WithContext(ctx).Warnf("settlement side effects failed for order %s: %v", order.ID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if duplicate {
		return &settlementOutcome{Duplicate: true}, nil
	}

	order.Status = string(StatusPaid)
	order.PayMethod = string(method)
	order.TransactionID = transactionID
	order.PaidAmount = order.FinalAmount

	uc.log.WithContext(ctx).Infof("payment settled: order=%s method=%s amount=%.2f transaction=%s",
		order.ID, method, order.FinalAmount, transactionID)
	return &settlementOutcome{}, nil
}

func (uc *PaymentUseCase) buildWechatErrorResponse(msg string) *v1.WechatCallbackResponse {
	return &v1.WechatCallbackResponse{
		ReturnCode: string(WechatStatusFail),
		ReturnMsg:  msg,
	}
}

func (uc *PaymentUseCase) buildAlipayErrorResponse(msg string) *v1.AlipayCallbackResponse {
	return &v1.AlipayCallbackResponse{
		Code: "FAIL",
		Msg:  msg,
	}
}

func isAlipaySuccessStatus(status string) bool {
	return status == string(AlipayStatusSuccess) || status == string(AlipayStatusFinished)
}

func (uc *PaymentUseCase) logSecurityEvent(ctx context.Context, eventType, orderID string, expected, received float64, transactionID string, details string) {
	uc.log.WithContext(ctx).Errorf("SECURITY EVENT [%s]: order=%s, expected=%.2f, received=%.2f, transaction=%s, details=%s",
		eventType, orderID, expected, received, transactionID, details)
}

// applyOrderSettledSideEffects runs the post-settlement side effect for an
// order: charging orders confirm their charging session, parking orders open
// the exit gate (after the record's exit_status is flipped to paid).
func (uc *PaymentUseCase) applyOrderSettledSideEffects(ctx context.Context, order *Order) error {
	if order.OrderType == OrderTypeCharging {
		return uc.confirmChargingSession(ctx, order)
	}
	return uc.triggerAutoGateOpen(ctx, order)
}

// confirmChargingSession marks the charging session paid through the charging
// service. For charging orders the RecordID carries the charging session ID
// (see CreatePayment).
func (uc *PaymentUseCase) confirmChargingSession(ctx context.Context, order *Order) error {
	if uc.chargingClient == nil {
		uc.log.WithContext(ctx).Warnf("charging service not configured, skipping payment confirmation for order %s", order.ID)
		return nil
	}

	if err := uc.chargingClient.ConfirmChargingPayment(ctx, order.RecordID.String(),
		order.TransactionID, order.PayMethod, order.PaidAmount); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to confirm charging session %s for order %s: %v",
			order.RecordID, order.ID, err)
		return fmt.Errorf("failed to confirm charging session: %w", err)
	}

	uc.log.WithContext(ctx).Infof("charging session %s confirmed for order %s", order.RecordID, order.ID)
	return nil
}

// triggerAutoGateOpen opens the exit gate once a payment is confirmed. A failure here is
// deliberately non-fatal: the order stays paid and the driver can still be released by
// re-scanning, whereas propagating the error would make the gateway retry a settlement
// that already succeeded.
func (uc *PaymentUseCase) triggerAutoGateOpen(ctx context.Context, order *Order) error {
	if uc.gateClient == nil || uc.recordRepo == nil {
		uc.log.WithContext(ctx).Warn("gate control service not configured, skipping auto gate open")
		return nil
	}

	record, err := uc.recordRepo.GetRecord(ctx, order.RecordID.String())
	if err != nil {
		return fmt.Errorf("failed to get record: %w", err)
	}
	// Repositories return (nil, nil) when the record simply does not exist; dereferencing
	// that would crash the callback handler and make the gateway retry forever.
	if record == nil {
		uc.log.WithContext(ctx).Warnf("parking record %s not found, skipping auto gate open", order.RecordID)
		return nil
	}

	if record.ExitDeviceID == "" {
		uc.log.WithContext(ctx).Info("no exit device ID found, skipping auto gate open")
		return nil
	}

	// 先回写支付状态再开闸：若开闸失败，司机在场内重新扫码时 Exit() 的
	// "已支付直接抬杆"快速路径也能命中，不会被要求二次支付。
	if err := uc.recordRepo.UpdateRecordStatus(ctx, record.ID, "paid"); err != nil {
		uc.log.WithContext(ctx).Warnf("failed to update record status: %v", err)
	}

	if err := uc.gateClient.OpenGate(ctx, record.ExitDeviceID, record.ID); err != nil {
		return fmt.Errorf("failed to open gate: %w", err)
	}

	uc.log.WithContext(ctx).Infof("auto gate opened successfully for record %s", record.ID)
	return nil
}

// yuanStringToCents converts an Alipay style "12.34" amount into integer cents without
// going through binary floating point, which cannot represent most decimal fractions.
func yuanStringToCents(amount string) (int64, error) {
	cents, err := parseYuanToCents(amount)
	if err != nil {
		return 0, err
	}
	return cents, nil
}
