// Package biz provides business logic for the payment service.
package biz

import (
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

const (
	SecurityEventAmountMismatch = "amount_mismatch"
	SecurityEventInvalidStatus  = "invalid_status"
)

type SecurityEvent struct {
	Type        string
	OrderID     string
	Expected    float64
	Received    float64
	Transaction string
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
	if req.ReturnCode != string(WechatStatusSuccess) {
		uc.log.WithContext(ctx).Warnf("WeChat callback failed: %s - %s", req.ReturnCode, req.ReturnMsg)
		return uc.buildWechatErrorResponse(req.ReturnMsg), nil
	}

	if err := uc.verifyWechatSign(req); err != nil {
		uc.log.WithContext(ctx).Errorf("WeChat signature verification failed: %v", err)
		return uc.buildWechatErrorResponse("Signature verification failed"), nil
	}

	if err := uc.processWechatPayment(ctx, req); err != nil {
		return uc.buildWechatErrorResponse(err.Error()), nil
	}

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
	// Look up order by OutTradeNo (order ID), not TransactionId
	// TransactionId is set only after payment is confirmed
	orderID, err := uuid.Parse(req.OutTradeNo)
	if err != nil {
		return fmt.Errorf("invalid out_trade_no: %w", err)
	}
	order, err := uc.orderRepo.GetOrder(ctx, orderID)
	if err != nil || order == nil {
		return fmt.Errorf("order not found: %s", req.OutTradeNo)
	}

	if order.Status != string(StatusPending) {
		uc.logSecurityEvent(ctx, SecurityEventInvalidStatus, order.ID.String(), 0, 0, req.TransactionId)
		return nil
	}

	paidAmount := parseAmount(req.TotalFee)
	if err := uc.validateAmount(order, paidAmount); err != nil {
		uc.logSecurityEvent(ctx, SecurityEventAmountMismatch, order.ID.String(), order.FinalAmount, paidAmount, req.TransactionId)
		return err
	}

	if err := uc.updateOrderAsPaid(order, MethodWechat, req.TransactionId, paidAmount); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	if err := uc.triggerAutoGateOpen(ctx, order); err != nil {
		uc.log.WithContext(ctx).Warnf("auto gate open failed: %v, owner can manually scan again", err)
	}

	return nil
}

func (uc *PaymentUseCase) validateAmount(order *Order, paidAmount float64) error {
	diff := math.Abs(paidAmount - order.FinalAmount)
	if diff > 0.01 {
		return fmt.Errorf("amount mismatch: expected %.2f, received %.2f", order.FinalAmount, paidAmount)
	}
	return nil
}

func (uc *PaymentUseCase) logSecurityEvent(ctx context.Context, eventType, orderID string, expected, received float64, transactionID string) {
	event := &SecurityEvent{
		Type:        eventType,
		OrderID:     orderID,
		Expected:    expected,
		Received:    received,
		Transaction: transactionID,
	}
	uc.log.WithContext(ctx).Errorf("security event: %+v", event)
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
		return fmt.Errorf("signature mismatch: expected %s, got %s", expectedSign, req.Sign)
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
	if !uc.isAlipaySuccessStatus(req.TradeStatus) {
		uc.log.WithContext(ctx).Warnf("Alipay callback failed: %s", req.TradeStatus)
		return uc.buildAlipayErrorResponse(req.TradeStatus), nil
	}

	if err := uc.verifyAlipaySign(req); err != nil {
		uc.log.WithContext(ctx).Errorf("Alipay signature verification failed: %v", err)
		return uc.buildAlipayErrorResponse("Signature verification failed"), nil
	}

	if err := uc.processAlipayPayment(ctx, req); err != nil {
		return uc.buildAlipayErrorResponse(err.Error()), nil
	}

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
	// Look up order by OutTradeNo (order ID), not TradeNo
	// TradeNo is the Alipay transaction ID, set only after payment
	orderID, err := uuid.Parse(req.OutTradeNo)
	if err != nil {
		return fmt.Errorf("invalid out_trade_no: %w", err)
	}
	order, err := uc.orderRepo.GetOrder(ctx, orderID)
	if err != nil || order == nil {
		return fmt.Errorf("order not found: %s", req.OutTradeNo)
	}

	if order.Status != string(StatusPending) {
		uc.logSecurityEvent(ctx, SecurityEventInvalidStatus, order.ID.String(), 0, 0, req.TradeNo)
		return nil
	}

	paidAmount := parseAmountFloat(req.TotalAmount)
	if err := uc.validateAmount(order, paidAmount); err != nil {
		uc.logSecurityEvent(ctx, SecurityEventAmountMismatch, order.ID.String(), order.FinalAmount, paidAmount, req.TradeNo)
		return err
	}

	if err := uc.updateOrderAsPaid(order, MethodAlipay, req.TradeNo, paidAmount); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	if err := uc.triggerAutoGateOpen(ctx, order); err != nil {
		uc.log.WithContext(ctx).Warnf("auto gate open failed: %v, owner can manually scan again", err)
	}

	return nil
}

func (uc *PaymentUseCase) updateOrderAsPaid(order *Order, method PayMethod, transactionID string, amount float64) error {
	if order.Status != string(StatusPending) {
		return fmt.Errorf("order status is not pending: %s", order.Status)
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
		return fmt.Errorf("signature verification failed: %w", err)
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

func hashData(h crypto.Hash, data string) []byte {
	hh := h.New()
	hh.Write([]byte(data))
	return hh.Sum(nil)
}
