package wechat

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Callback verification errors. Every one of them means the notification must be rejected.
var (
	ErrMissingPlatformCert = errors.New("wechat: platform certificate is not configured")
	ErrMissingAPIv3Key     = errors.New("wechat: api_v3_key is not configured")
	ErrInvalidSignature    = errors.New("wechat: callback signature verification failed")
	ErrSerialMismatch      = errors.New("wechat: callback certificate serial does not match the configured platform certificate")
	ErrCallbackTooOld      = errors.New("wechat: callback timestamp is outside the accepted replay window")
)

// CallbackReplayWindow bounds how far a WeChat callback timestamp may drift from server time.
// It protects against replaying a previously captured (validly signed) notification.
const CallbackReplayWindow = 5 * time.Minute

// Notification is the envelope of a WeChat Pay APIv3 callback.
type Notification struct {
	ID           string      `json:"id"`
	CreateTime   string      `json:"create_time"`
	EventType    string      `json:"event_type"`
	ResourceType string      `json:"resource_type"`
	Summary      string      `json:"summary"`
	Resource     Resource    `json:"resource"`
}

// Resource is the encrypted payload carried by an APIv3 notification.
type Resource struct {
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	Nonce          string `json:"nonce"`
	AssociatedData string `json:"associated_data"`
}

// TransactionResource is the decrypted body of a payment/refund notification.
type TransactionResource struct {
	AppID         string `json:"appid"`
	MchID         string `json:"mchid"`
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeType     string `json:"trade_type"`
	TradeState    string `json:"trade_state"`
	SuccessTime   string `json:"success_time"`
	OutRefundNo   string `json:"out_refund_no"`
	RefundID      string `json:"refund_id"`
	RefundStatus  string `json:"refund_status"`
	Amount        struct {
		Total       int64  `json:"total"`
		PayerTotal  int64  `json:"payer_total"`
		Refund      int64  `json:"refund"`
		PayerRefund int64  `json:"payer_refund"`
		Currency    string `json:"currency"`
	} `json:"amount"`
}

// TradeStateSuccess is the only trade_state that represents a completed payment.
const TradeStateSuccess = "SUCCESS"

// CallbackEventTypePayment is the event_type of a successful payment notification.
const CallbackEventTypePayment = "TRANSACTION.SUCCESS"

// VerifyCallbackSignature verifies the RSA-PSS signature of an APIv3 callback.
//
// The signed message is exactly `timestamp\nnonce\nbody\n` per the APIv3 specification.
// serial identifies the platform certificate that produced the signature and must match
// the certificate we have on file, otherwise an attacker could sign with any certificate
// that happens to be trusted.
func (c *Client) VerifyCallbackSignature(serial, timestamp, nonce, body, signature string) error {
	cert := c.platformCert
	if cert == nil {
		return ErrMissingPlatformCert
	}
	if serial == "" {
		return fmt.Errorf("%w: empty Wechatpay-Serial header", ErrInvalidSignature)
	}
	if serial != c.config.CertSerialNo {
		return fmt.Errorf("%w: header %q, configured %q", ErrSerialMismatch, serial, c.config.CertSerialNo)
	}

	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("wechat: platform certificate does not carry an RSA public key")
	}

	raw, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("%w: signature is not valid base64: %w", ErrInvalidSignature, err)
	}

	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, body)
	digest := sha256.Sum256([]byte(message))

	if err := rsa.VerifyPSS(pub, crypto.SHA256, digest[:], raw, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSignature, err)
	}
	return nil
}

// VerifyCallbackTimestamp rejects callbacks whose timestamp falls outside the replay window.
// now is injected so the check is testable without sleeping.
func VerifyCallbackTimestamp(timestamp string, now time.Time) error {
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("wechat: invalid Wechatpay-Timestamp header %q: %w", timestamp, err)
	}
	ts := time.Unix(sec, 0)
	delta := now.Sub(ts)
	if delta < -CallbackReplayWindow || delta > CallbackReplayWindow {
		return fmt.Errorf("%w: callback at %s, server at %s", ErrCallbackTooOld, ts.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	return nil
}

// DecryptResource decrypts the APIv3 notification payload with AES-256-GCM.
func (c *Client) DecryptResource(res Resource) (*TransactionResource, error) {
	if c.config.APIv3Key == "" {
		return nil, ErrMissingAPIv3Key
	}
	if res.Algorithm != "AEAD_AES_256_GCM" {
		return nil, fmt.Errorf("wechat: unsupported resource algorithm %q", res.Algorithm)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(res.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("wechat: ciphertext is not valid base64: %w", err)
	}

	block, err := aes.NewCipher([]byte(c.config.APIv3Key))
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to build AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to build GCM: %w", err)
	}
	if len(res.Nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("wechat: invalid nonce length %d, want %d", len(res.Nonce), gcm.NonceSize())
	}

	plain, err := gcm.Open(nil, []byte(res.Nonce), ciphertext, []byte(res.AssociatedData))
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to decrypt callback resource: %w", err)
	}

	var out TransactionResource
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, fmt.Errorf("wechat: failed to decode decrypted callback resource: %w", err)
	}
	return &out, nil
}

// ParseAndVerifyCallback performs the full APIv3 callback validation pipeline:
// timestamp freshness -> signature -> decryption. It returns the decrypted transaction
// only when every step succeeds.
func (c *Client) ParseAndVerifyCallback(headers map[string]string, body []byte, now time.Time) (*TransactionResource, error) {
	serial := headers["Wechatpay-Serial"]
	timestamp := headers["Wechatpay-Timestamp"]
	nonce := headers["Wechatpay-Nonce"]
	signature := headers["Wechatpay-Signature"]

	if serial == "" || timestamp == "" || nonce == "" || signature == "" {
		return nil, fmt.Errorf("%w: missing required Wechatpay-* headers", ErrInvalidSignature)
	}

	if err := VerifyCallbackTimestamp(timestamp, now); err != nil {
		return nil, err
	}
	if err := c.VerifyCallbackSignature(serial, timestamp, nonce, string(body), signature); err != nil {
		return nil, err
	}

	var notification Notification
	if err := json.Unmarshal(body, &notification); err != nil {
		return nil, fmt.Errorf("wechat: failed to decode callback body: %w", err)
	}

	return c.DecryptResource(notification.Resource)
}
