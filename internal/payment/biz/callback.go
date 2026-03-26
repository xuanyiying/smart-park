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
	"sort"
	"strings"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

// HandleWechatCallback handles WeChat payment callback.
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

// buildWechatErrorResponse builds WeChat error response.
func (uc *PaymentUseCase) buildWechatErrorResponse(msg string) *v1.WechatCallbackResponse {
	return &v1.WechatCallbackResponse{
		ReturnCode: string(WechatStatusFail),
		ReturnMsg:  msg,
	}
}

// processWechatPayment processes WeChat payment callback.
func (uc *PaymentUseCase) processWechatPayment(ctx context.Context, req *v1.WechatCallbackRequest) error {
	order, err := uc.orderRepo.GetOrderByTransactionID(ctx, req.TransactionId)
	if err != nil {
		return fmt.Errorf("order not found")
	}

	if err := uc.updateOrderAsPaid(order, MethodWechat, req.TransactionId, parseAmount(req.TotalFee)); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	return nil
}

// verifyWechatSign verifies WeChat Pay callback signature.
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

// buildWechatSignString builds the string to sign for WeChat Pay.
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

// HandleAlipayCallback handles Alipay payment callback.
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

// isAlipaySuccessStatus checks if the Alipay status indicates success.
func (uc *PaymentUseCase) isAlipaySuccessStatus(status string) bool {
	return status == string(AlipayStatusSuccess) || status == string(AlipayStatusFinished)
}

// buildAlipayErrorResponse builds Alipay error response.
func (uc *PaymentUseCase) buildAlipayErrorResponse(msg string) *v1.AlipayCallbackResponse {
	return &v1.AlipayCallbackResponse{
		Code: "FAIL",
		Msg:  msg,
	}
}

// processAlipayPayment processes Alipay payment callback.
func (uc *PaymentUseCase) processAlipayPayment(ctx context.Context, req *v1.AlipayCallbackRequest) error {
	order, err := uc.orderRepo.GetOrderByTransactionID(ctx, req.TradeNo)
	if err != nil {
		return fmt.Errorf("order not found")
	}

	if err := uc.updateOrderAsPaid(order, MethodAlipay, req.TradeNo, parseAmountFloat(req.TotalAmount)); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return fmt.Errorf("update failed")
	}

	return nil
}

// updateOrderAsPaid updates order status to paid.
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

// verifyAlipaySign verifies Alipay callback signature using RSA.
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

// parseAlipayPublicKey parses Alipay public key from PEM.
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

// getAlipayHashAlgorithm returns the hash algorithm based on key type.
func (uc *PaymentUseCase) getAlipayHashAlgorithm() crypto.Hash {
	if strings.Contains(uc.config.AlipayPublicKey, "RSA2") {
		return crypto.SHA256
	}
	return crypto.SHA1
}

// buildAlipaySignString builds the string to sign for Alipay.
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

// buildSignString builds sign string from fields, excluding specified key.
func buildSignString(fields map[string]string, excludeKey string) string {
	var keys []string
	for k := range fields {
		if k != excludeKey && fields[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(fields[k])
		sb.WriteString("&")
	}
	return sb.String()
}

// buildSignStringWithAmpersand builds sign string with & separator.
func buildSignStringWithAmpersand(fields map[string]string) string {
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if fields[k] != "" {
			if i > 0 {
				sb.WriteString("&")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(fields[k])
		}
	}
	return sb.String()
}

// calculateMD5 calculates MD5 hash.
func calculateMD5(input string) string {
	h := md5.New()
	h.Write([]byte(input))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// hashData computes hash of data.
func hashData(h crypto.Hash, data string) []byte {
	hh := h.New()
	hh.Write([]byte(data))
	return hh.Sum(nil)
}
