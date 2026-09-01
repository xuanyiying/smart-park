package alipay

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Callback verification errors. Every one of them means the notification must be rejected.
var (
	ErrMissingPublicKey = errors.New("alipay: public key is not configured")
	ErrInvalidSignature = errors.New("alipay: callback signature verification failed")
	ErrMissingSign      = errors.New("alipay: callback does not carry a sign parameter")
)

// SignTypeRSA / SignTypeRSA2 are the two signature algorithms supported by Alipay.
const (
	SignTypeRSA  = "RSA"
	SignTypeRSA2 = "RSA2"
)

// NormalizeSignType maps a configured sign type onto the canonical value, defaulting to
// RSA2 (SHA256) which is what all modern Alipay apps use.
func NormalizeSignType(signType string) string {
	switch strings.ToUpper(strings.TrimSpace(signType)) {
	case SignTypeRSA:
		return SignTypeRSA
	case SignTypeRSA2:
		return SignTypeRSA2
	default:
		return SignTypeRSA2
	}
}

// hashForSignType resolves the digest used by the configured signature algorithm.
// The mapping is driven by an explicit configuration value, never by inspecting the
// key material, which makes it deterministic and testable.
func hashForSignType(signType string) crypto.Hash {
	if NormalizeSignType(signType) == SignTypeRSA {
		return crypto.SHA1
	}
	return crypto.SHA256
}

// BuildSignContent assembles the string that Alipay signs, from the *complete* set of
// callback parameters. `sign` and `sign_type` are excluded, remaining keys are sorted by
// ASCII order and joined with '&' as `k=v`.
//
// Signing only a subset of parameters (as an earlier implementation did) both breaks
// verification and lets an attacker tamper with the fields that were left out.
func BuildSignContent(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params.Get(k))
	}
	return sb.String()
}

// parsePublicKey accepts either a PEM public key or a bare base64 DER blob, which is how
// Alipay presents keys in its console.
func parsePublicKey(publicKey string) (*rsa.PublicKey, error) {
	trimmed := strings.TrimSpace(publicKey)
	if trimmed == "" {
		return nil, ErrMissingPublicKey
	}

	if block, _ := pem.Decode([]byte(trimmed)); block != nil {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("alipay: failed to parse public key: %w", err)
		}
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("alipay: public key is not an RSA key")
		}
		return rsaPub, nil
	}

	der, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("alipay: public key is neither PEM nor base64 DER: %w", err)
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("alipay: public key is not an RSA key")
	}
	return rsaPub, nil
}

// VerifyNotification verifies an Alipay callback against the merchant's copy of the
// Alipay public key, using every received parameter.
func (c *Client) VerifyNotification(params url.Values) error {
	if c.config.AlipayPublicKey == "" {
		return ErrMissingPublicKey
	}

	sign := params.Get("sign")
	if sign == "" {
		return ErrMissingSign
	}

	// The sign_type parameter is itself part of the signed payload in Alipay's scheme only
	// when it is absent from the exclusion list; we always exclude it, and fall back to the
	// configured algorithm when the caller did not send one.
	signType := params.Get("sign_type")
	if signType == "" {
		signType = c.config.SignType
	}

	pub, err := parsePublicKey(c.config.AlipayPublicKey)
	if err != nil {
		return err
	}

	raw, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("%w: sign is not valid base64: %w", ErrInvalidSignature, err)
	}

	content := BuildSignContent(params)
	h := hashForSignType(signType)
	digest := h.New()
	digest.Write([]byte(content))

	if err := rsa.VerifyPKCS1v15(pub, h, digest.Sum(nil), raw); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSignature, err)
	}
	return nil
}

// ParseNotification verifies the signature and returns the parameters, refusing to hand
// unverified data back to the caller.
func (c *Client) ParseNotification(params url.Values) (url.Values, error) {
	if err := c.VerifyNotification(params); err != nil {
		return nil, err
	}
	return params, nil
}
