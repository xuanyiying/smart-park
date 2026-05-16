package biz

import (
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

const (
	SecurityEventAmountMismatch    = "amount_mismatch"
	SecurityEventInvalidStatus     = "invalid_status"
	SecurityEventDuplicateCallback = "duplicate_callback"
	SecurityEventSignatureFailed   = "signature_failed"
	SecurityEventInvalidTradeNo    = "invalid_trade_no"
	SecurityEventOrderNotFound     = "order_not_found"
	SecurityEventUnknown           = "unknown"
)

var (
	processedCallbacks = &callbackDeduplication{
		callbacks: make(map[string]time.Time),
	}
)

type callbackDeduplication struct {
	mu        sync.RWMutex
	callbacks map[string]time.Time
}

func (d *callbackDeduplication) isProcessed(id string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, exists := d.callbacks[id]
	return exists
}

func (d *callbackDeduplication) markProcessed(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.callbacks[id] = time.Now()
	d.cleanup()
}

func (d *callbackDeduplication) cleanup() {
	now := time.Now()
	for id, timestamp := range d.callbacks {
		if now.Sub(timestamp) > 24*time.Hour {
			delete(d.callbacks, id)
		}
	}
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
}

func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
	callbackID := fmt.Sprintf("wechat_%s_%s", req.OutTradeNo, req.TransactionId)

	if processedCallbacks.isProcessed(callbackID) {
		uc.log.WithContext(ctx).Warnf("duplicate WeChat callback detected: %s", callbackID)
		return &v1.WechatCallbackResponse{
			ReturnCode: string(WechatStatusSuccess),
			ReturnMsg:  "OK",
		}, nil
	}

	if req.ReturnCode != string(WechatStatusSuccess) {
		uc.log.WithContext(ctx).Warnf("WeChat callback failed: %s - %s", req.ReturnCode, req.ReturnMsg)
		return uc.buildWechatErrorResponse(req.ReturnMsg), nil
	}

	if err := uc.verifyWechatSign(req); err != nil {
		uc.logSecurityEvent(ctx, SecurityEventSignatureFailed, req.OutTradeNo, 0, parseAmount(req.TotalFee), req.TransactionId, err.Error())
		return uc.buildWechatErrorResponse("Signature verification failed"), nil
	}

	if err := uc.validateWechatCallbackTime(req.TimeEnd); err != nil {
		uc.log.WithContext(ctx).Warnf("WeChat callback time validation failed: %v", err)
	}

	if err := uc.processWechatPayment(ctx, req); err != nil {
		return uc.buildWechatErrorResponse(err.Error()), nil
	}

	processedCallbacks.markProcessed(callbackID)

	return &v1.WechatCallbackResponse{
		ReturnCode: string(WechatStatusSuccess),
		ReturnMsg:  "OK",
	}, nil
}

func (uc *PaymentUseCase) buildWechatErrorResponse(msg string) *v1.WechatCallbackResponse {
	return &v1.WechatCallbackResponse{
		ReturnCode: string(WechatStatusFail),
		ReturnMsg:  msg,
	}
}

func (uc *PaymentUseCase) processWechatPayment(ctx context.Context, req *v1.WechatCallbackRequest) error {
	orderID, err := uuid.Parse(req.OutTradeNo)
	if err != nil {
		uc.logSecurityEvent(ctx, SecurityEventInvalidTradeNo, req.OutTradeNo, 0, parseAmount(req.TotalFee), req.TransactionId, err.Error())
		return fmt.Errorf("invalid out_trade_no: %w", err)
	}

	var order *Order

	err = uc.orderRepo.WithTx(ctx, func(txCtx context.Context) error {
		var err error
		order, err = uc.orderRepo.GetOrder(txCtx, orderID)
		if err != nil || order == nil {
			uc.logSecurityEvent(txCtx, SecurityEventOrderNotFound, req.OutTradeNo, 0, parseAmount(req.TotalFee), req.TransactionId, "order not found")
			return fmt.Errorf("order not found: %s", req.OutTradeNo)
		}

		if order.Status != string(StatusPending) {
			uc.logSecurityEvent(txCtx, SecurityEventInvalidStatus, order.ID.String(), order.FinalAmount, parseAmount(req.TotalFee), req.TransactionId,
				fmt.Sprintf("status: %s", order.Status))
			return nil
		}

		paidAmount := parseAmount(req.TotalFee)
		if err := uc.validateAmount(order, paidAmount); err != nil {
			uc.logSecurityEvent(txCtx, SecurityEventAmountMismatch, order.ID.String(), order.FinalAmount, paidAmount, req.TransactionId, err.Error())
			return err
		}

		if err := uc.updateOrderAsPaid(order, MethodWechat, req.TransactionId, paidAmount); err != nil {
			uc.log.WithContext(txCtx).Errorf("failed to update order: %v", err)
			return fmt.Errorf("update failed")
		}

		return uc.orderRepo.UpdateOrder(txCtx, order)
	})
	if err != nil {
		return err
	}

	if order != nil && order.Status == string(StatusPaid) {
		if err := uc.triggerAutoGateOpen(ctx, order); err != nil {
			uc.log.WithContext(ctx).Warnf("auto gate open failed: %v, owner can manually scan again", err)
		}
	}

	uc.log.WithContext(ctx).Infof("WeChat payment processed successfully: order=%s, amount=%.2f, transaction=%s",
		order.ID.String(), parseAmount(req.TotalFee), req.TransactionId)

	return nil
}

func (uc *PaymentUseCase) validateAmount(order *Order, paidAmount float64) error {
	if paidAmount < 0 {
		return fmt.Errorf("invalid paid amount: %.2f", paidAmount)
	}

	diff := math.Abs(paidAmount - order.FinalAmount)
	if diff > 0.01 {
		return fmt.Errorf("amount mismatch: expected %.2f, received %.2f", order.FinalAmount, paidAmount)
	}
	return nil
}

func (uc *PaymentUseCase) logSecurityEvent(ctx context.Context, eventType, orderID string, expected, received float64, transactionID string, details string) {
	uc.log.WithContext(ctx).Errorf("SECURITY EVENT [%s]: order=%s, expected=%.2f, received=%.2f, transaction=%s, details=%s",
		eventType, orderID, expected, received, transactionID, details)
}

func (uc *PaymentUseCase) triggerAutoGateOpen(ctx context.Context, order *Order) error {
	if uc.gateClient == nil || uc.recordRepo == nil {
		uc.log.WithContext(ctx).Warn("gate control service not configured, skipping auto gate open")
		return nil
	}

	record, err := uc.recordRepo.GetRecord(ctx, order.RecordID.String())
	if err != nil {
		return fmt.Errorf("failed to get record: %w", err)
	}

	if record.ExitDeviceID == "" {
		uc.log.WithContext(ctx).Info("no exit device ID found, skipping auto gate open")
		return nil
	}

	if err := uc.gateClient.OpenGate(ctx, record.ExitDeviceID, record.ID); err != nil {
		return fmt.Errorf("failed to open gate: %w", err)
	}

	if err := uc.recordRepo.UpdateRecordStatus(ctx, record.ID, "paid"); err != nil {
		uc.log.WithContext(ctx).Warnf("failed to update record status: %v", err)
	}

	uc.log.WithContext(ctx).Infof("auto gate opened successfully for record %s", record.ID)
	return nil
}

func (uc *PaymentUseCase) verifyWechatSign(req *v1.WechatCallbackRequest) error {
	if uc.config == nil || uc.config.WechatKey == "" {
		return fmt.Errorf("wechat key not configured")
	}

	signData := buildWechatSignString(req)
	expectedSign := calculateMD5(signData + "&key=" + uc.config.WechatKey)

	if !strings.EqualFold(req.Sign, expectedSign) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func (uc *PaymentUseCase) validateWechatCallbackTime(timeEnd string) error {
	if timeEnd == "" {
		return fmt.Errorf("time_end is empty")
	}

	callbackTime, err := time.ParseInLocation("20060102150405", timeEnd, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	now := time.Now()
	if callbackTime.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("callback time is in the future")
	}

	if callbackTime.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("callback time is too old (>24h)")
	}

	return nil
}

func buildWechatSignString(req *v1.WechatCallbackRequest) string {
	fields := map[string]string{
		"return_code":    req.ReturnCode,
		"return_msg":     req.ReturnMsg,
		"result_code":    req.ResultCode,
		"transaction_id": req.TransactionId,
		"out_trade_no":   req.OutTradeNo,
		"total_fee":      req.TotalFee,
		"time_end":       req.TimeEnd,
	}

	return buildSignString(fields, "sign")
}

func (uc *PaymentUseCase) HandleAlipayCallback(ctx context.Context, req *v1.AlipayCallbackRequest) (*v1.AlipayCallbackResponse, error) {
	callbackID := fmt.Sprintf("alipay_%s_%s", req.OutTradeNo, req.TradeNo)

	if processedCallbacks.isProcessed(callbackID) {
		uc.log.WithContext(ctx).Warnf("duplicate Alipay callback detected: %s", callbackID)
		return &v1.AlipayCallbackResponse{
			Code: "success",
			Msg:  "OK",
		}, nil
	}

	if !uc.isAlipaySuccessStatus(req.TradeStatus) {
		uc.log.WithContext(ctx).Warnf("Alipay callback failed: %s", req.TradeStatus)
		return uc.buildAlipayErrorResponse(req.TradeStatus), nil
	}

	if err := uc.verifyAlipaySign(req); err != nil {
		uc.logSecurityEvent(ctx, SecurityEventSignatureFailed, req.OutTradeNo, 0, parseAmountFloat(req.TotalAmount), req.TradeNo, err.Error())
		return uc.buildAlipayErrorResponse("Signature verification failed"), nil
	}

	if err := uc.validateAlipayCallbackTime(req.GmtPayment); err != nil {
		uc.log.WithContext(ctx).Warnf("Alipay callback time validation failed: %v", err)
	}

	if err := uc.processAlipayPayment(ctx, req); err != nil {
		return uc.buildAlipayErrorResponse(err.Error()), nil
	}

	processedCallbacks.markProcessed(callbackID)

	return &v1.AlipayCallbackResponse{
		Code: "success",
		Msg:  "OK",
	}, nil
}

func (uc *PaymentUseCase) isAlipaySuccessStatus(status string) bool {
	return status == string(AlipayStatusSuccess) || status == string(AlipayStatusFinished)
}

func (uc *PaymentUseCase) buildAlipayErrorResponse(msg string) *v1.AlipayCallbackResponse {
	return &v1.AlipayCallbackResponse{
		Code: "FAIL",
		Msg:  msg,
	}
}

func (uc *PaymentUseCase) processAlipayPayment(ctx context.Context, req *v1.AlipayCallbackRequest) error {
	orderID, err := uuid.Parse(req.OutTradeNo)
	if err != nil {
		uc.logSecurityEvent(ctx, SecurityEventInvalidTradeNo, req.OutTradeNo, 0, parseAmountFloat(req.TotalAmount), req.TradeNo, err.Error())
		return fmt.Errorf("invalid out_trade_no: %w", err)
	}

	var order *Order

	err = uc.orderRepo.WithTx(ctx, func(txCtx context.Context) error {
		var err error
		order, err = uc.orderRepo.GetOrder(txCtx, orderID)
		if err != nil || order == nil {
			uc.logSecurityEvent(txCtx, SecurityEventOrderNotFound, req.OutTradeNo, 0, parseAmountFloat(req.TotalAmount), req.TradeNo, "order not found")
			return fmt.Errorf("order not found: %s", req.OutTradeNo)
		}

		if order.Status != string(StatusPending) {
			uc.logSecurityEvent(txCtx, SecurityEventInvalidStatus, order.ID.String(), order.FinalAmount, parseAmountFloat(req.TotalAmount), req.TradeNo,
				fmt.Sprintf("status: %s", order.Status))
			return nil
		}

		paidAmount := parseAmountFloat(req.TotalAmount)
		if err := uc.validateAmount(order, paidAmount); err != nil {
			uc.logSecurityEvent(txCtx, SecurityEventAmountMismatch, order.ID.String(), order.FinalAmount, paidAmount, req.TradeNo, err.Error())
			return err
		}

		if err := uc.updateOrderAsPaid(order, MethodAlipay, req.TradeNo, paidAmount); err != nil {
			uc.log.WithContext(txCtx).Errorf("failed to update order: %v", err)
			return fmt.Errorf("update failed")
		}

		return uc.orderRepo.UpdateOrder(txCtx, order)
	})
	if err != nil {
		return err
	}

	if order != nil && order.Status == string(StatusPaid) {
		if err := uc.triggerAutoGateOpen(ctx, order); err != nil {
			uc.log.WithContext(ctx).Warnf("auto gate open failed: %v, owner can manually scan again", err)
		}
	}

	uc.log.WithContext(ctx).Infof("Alipay payment processed successfully: order=%s, amount=%.2f, trade_no=%s",
		order.ID.String(), parseAmountFloat(req.TotalAmount), req.TradeNo)

	return nil
}

func (uc *PaymentUseCase) validateAlipayCallbackTime(gmtPayment string) error {
	if gmtPayment == "" {
		return fmt.Errorf("gmt_payment is empty")
	}

	callbackTime, err := time.ParseInLocation("2006-01-02 15:04:05", gmtPayment, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	now := time.Now()
	if callbackTime.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("callback time is in the future")
	}

	if callbackTime.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("callback time is too old (>24h)")
	}

	return nil
}

func (uc *PaymentUseCase) updateOrderAsPaid(order *Order, method PayMethod, transactionID string, amount float64) error {
	if order.Status != string(StatusPending) {
		return fmt.Errorf("order status is not pending: %s", order.Status)
	}

	if amount <= 0 {
		return fmt.Errorf("invalid payment amount: %.2f", amount)
	}

	now := currentTime()
	order.Status = string(StatusPaid)
	order.PayTime = &now
	order.PayMethod = string(method)
	order.TransactionID = transactionID
	order.PaidAmount = amount
	return nil
}

func (uc *PaymentUseCase) verifyAlipaySign(req *v1.AlipayCallbackRequest) error {
	if uc.config == nil || uc.config.AlipayPublicKey == "" {
		return fmt.Errorf("alipay public key not configured")
	}

	signData := buildAlipaySignString(req)

	pubKey, err := uc.parseAlipayPublicKey()
	if err != nil {
		return err
	}

	signBytes, err := base64.StdEncoding.DecodeString(req.Sign)
	if err != nil {
		return fmt.Errorf("failed to decode sign: %w", err)
	}

	hash := uc.getAlipayHashAlgorithm()

	if err := rsa.VerifyPKCS1v15(pubKey, hash, hashData(hash, signData), signBytes); err != nil {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

func (uc *PaymentUseCase) parseAlipayPublicKey() (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(uc.config.AlipayPublicKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return pubKey, nil
}

func (uc *PaymentUseCase) getAlipayHashAlgorithm() crypto.Hash {
	if strings.Contains(uc.config.AlipayPublicKey, "RSA2") {
		return crypto.SHA256
	}
	return crypto.SHA1
}

func buildAlipaySignString(req *v1.AlipayCallbackRequest) string {
	fields := map[string]string{
		"trade_status": req.TradeStatus,
		"trade_no":     req.TradeNo,
		"out_trade_no": req.OutTradeNo,
		"total_amount": req.TotalAmount,
		"gmt_payment":  req.GmtPayment,
	}

	return buildSignStringWithAmpersand(fields)
}

func buildSignString(fields map[string]string, excludeKey string) string {
	var keys []string
	for k := range fields {
		if k != excludeKey && fields[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(fields[k])
	}
	return sb.String()
}

func buildSignStringWithAmpersand(fields map[string]string) string {
	var keys []string
	for k := range fields {
		if fields[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(fields[k])
	}
	return sb.String()
}

func calculateMD5(input string) string {
	h := md5.New()
	h.Write([]byte(input))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

func calculateSHA256(input string) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

func hashData(h crypto.Hash, data string) []byte {
	hh := h.New()
	hh.Write([]byte(data))
	return hh.Sum(nil)
}
