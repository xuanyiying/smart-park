package biz

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Email transport
// ---------------------------------------------------------------------------

// smtpSession is the subset of *smtp.Client used for delivery, so tests can
// substitute a recording double without running a real server.
type smtpSession interface {
	Hello(string) error
	StartTLS(*tls.Config) error
	Auth(smtp.Auth) error
	Mail(string) error
	Rcpt(string) error
	Data() (io.WriteCloser, error)
	Quit() error
}

// dialSMTPFn is the dial used by deliverEmail. It is a variable so tests can
// substitute a recording session; production code never touches it.
var dialSMTPFn = dialSMTP

// smsPostFn performs the signed gateway request. Variable for the same reason.
var smsPostFn = func(ctx context.Context, endpoint, form string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return http.DefaultClient.Do(req)
}

// dialSMTP opens a session. Implicit TLS (usually port 465) dials through TLS
// directly; everything else dials plaintext and upgrades with STARTTLS when the
// server advertises it.
func dialSMTP(cfg *EmailConfig) (smtpSession, error) {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	tlsCfg := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.UseTLS {
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return nil, fmt.Errorf("smtp: tls dial %s: %w", addr, err)
		}
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("smtp: new client: %w", err)
		}
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("smtp: dial %s: %w", addr, err)
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsCfg); err != nil {
			client.Close()
			return nil, fmt.Errorf("smtp: starttls: %w", err)
		}
	}
	return client, nil
}

// deliverEmail authenticates and sends one message.
func deliverEmail(cfg *EmailConfig, to string, message []byte) error {
	session, err := dialSMTPFn(cfg)
	if err != nil {
		return err
	}
	defer session.Quit()

	if err := session.Hello(cfg.Host); err != nil {
		return fmt.Errorf("smtp: hello: %w", err)
	}
	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := session.Auth(auth); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}
	if err := session.Mail(cfg.From); err != nil {
		return fmt.Errorf("smtp: mail from: %w", err)
	}
	if err := session.Rcpt(to); err != nil {
		return fmt.Errorf("smtp: rcpt to: %w", err)
	}
	writer, err := session.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("smtp: write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp: close body: %w", err)
	}
	return nil
}

// buildMIMEMessage renders a minimal RFC 5322 message. Subject and body are
// quoted-printable safe here because headers are folded and the body is sent as
// UTF-8 with an explicit charset.
func buildMIMEMessage(from, to, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + mimeEncodeHeader(subject) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	sb.WriteString("\r\n")
	return []byte(sb.String())
}

// mimeEncodeHeader encodes a header value that may contain non-ASCII text,
// which otherwise arrives as mojibake in most mail clients.
func mimeEncodeHeader(s string) string {
	if s == "" || isASCII(s) {
		return s
	}
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7f {
			return false
		}
	}
	return true
}

// deliveryTimeout bounds a single outbound delivery so a stalled provider
// cannot hold a request goroutine indefinitely.
const deliveryTimeout = 15 * time.Second

// ---------------------------------------------------------------------------
// SMS transport (Aliyun compatible signed request)
// ---------------------------------------------------------------------------

type smsResponse struct {
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestID string `json:"RequestId"`
	BizID     string `json:"BizId"`
}

// sendSMS posts a signed SendSms request to the configured gateway.
func sendSMS(ctx context.Context, cfg *SMSConfig, phone string, params map[string]string) error {
	templateParam := "{}"
	if len(params) > 0 {
		encoded, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("sms: encode template params: %w", err)
		}
		templateParam = string(encoded)
	}

	values := url.Values{
		"AccessKeyId":      {cfg.AccessKeyID},
		"Action":           {"SendSms"},
		"Format":           {"JSON"},
		"PhoneNumbers":     {phone},
		"RegionId":         {"cn-hangzhou"},
		"SignName":         {cfg.SignName},
		"SignatureMethod":  {"HMAC-SHA1"},
		"SignatureNonce":   {uuid.New().String()},
		"SignatureVersion": {"1.0"},
		"TemplateCode":     {cfg.TemplateCode},
		"TemplateParam":    {templateParam},
		"Timestamp":        {time.Now().UTC().Format("2006-01-02T15:04:05Z")},
		"Version":          {"2017-05-25"},
	}
	values.Set("Signature", signPOPRequest("POST", cfg.AccessKey, values))

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = deliveryTimeout
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := smsPostFn(reqCtx, cfg.EndpointURL(), values.Encode())
	if err != nil {
		return fmt.Errorf("sms: send: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("sms: read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sms: gateway returned %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed smsResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("sms: decode response: %w", err)
	}
	// "OK" is the only success code; anything else (invalid template, insufficient
	// balance, blocked number) must surface as an error, not a silent success.
	if parsed.Code != "OK" {
		return fmt.Errorf("sms: gateway rejected request: %s %s (request %s)", parsed.Code, parsed.Message, parsed.RequestID)
	}
	return nil
}

// signPOPRequest builds the Alibaba Cloud POP signature:
// HMAC-SHA1 over "METHOD&%2F&<percent-encoded canonical query>".
func signPOPRequest(method, accessSecret string, values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for i, k := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(popPercentEncode(k))
		canonical.WriteByte('=')
		canonical.WriteString(popPercentEncode(values.Get(k)))
	}

	stringToSign := method + "&" + popPercentEncode("/") + "&" + popPercentEncode(canonical.String())

	mac := hmac.New(sha1.New, []byte(accessSecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// popPercentEncode follows the POP rules: URL encode, then restore the
// characters the gateway expects to see unescaped.
func popPercentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// ---------------------------------------------------------------------------
// WeChat transport (Official Account template message)
// ---------------------------------------------------------------------------

type wechatTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type wechatSendResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	MsgID   string `json:"msgid"`
}

// wechatToken caches the Official Account access token. WeChat allows a limited
// number of refreshes per day, so every notification must reuse one token until
// it is about to expire.
type wechatToken struct {
	value     string
	expiresAt time.Time
}

func (t *wechatToken) valid(now time.Time) bool {
	return t.value != "" && now.Before(t.expiresAt)
}

// wechatTokenCache holds one token per notifier. WeChat caps how often a token
// may be refreshed, so the cache is shared by every send on that notifier.
type wechatTokenCache struct {
	mu    sync.Mutex
	token *wechatToken
}

// fetchAccessToken retrieves (and caches) an access token for the account.
func (c *wechatTokenCache) fetchAccessToken(ctx context.Context, cfg *WechatConfig) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached := c.token; cached != nil && cached.valid(time.Now()) {
		return cached.value, nil
	}

	endpoint := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		cfg.Base(), url.QueryEscape(cfg.AppID), url.QueryEscape(cfg.Secret))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("wechat: build token request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wechat: token request: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("wechat: read token response: %w", err)
	}

	var parsed wechatTokenResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", fmt.Errorf("wechat: decode token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("wechat: token request failed: %d %s", parsed.ErrCode, parsed.ErrMsg)
	}

	ttl := time.Duration(parsed.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	// Renew a minute early so a token does not expire mid-send.
	c.token = &wechatToken{value: parsed.AccessToken, expiresAt: time.Now().Add(ttl - time.Minute)}
	return parsed.AccessToken, nil
}

// sendWechatTemplate posts a template message to the given OpenID.
func sendWechatTemplate(ctx context.Context, cfg *WechatConfig, tokens *wechatTokenCache, openID, title, content string) error {
	if openID == "" {
		return fmt.Errorf("wechat: recipient has no openid")
	}

	accessToken, err := tokens.fetchAccessToken(ctx, cfg)
	if err != nil {
		return err
	}

	// Template bodies are defined in the Official Account console; first,
	// keyword1 and remark are the conventional slots those templates expose.
	body := map[string]any{
		"touser":      openID,
		"template_id": cfg.TemplateID,
		"data": map[string]map[string]string{
			"first":    {"value": title},
			"keyword1": {"value": content},
			"remark":   {"value": time.Now().Format("2006-01-02 15:04")},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("wechat: encode message: %w", err)
	}

	endpoint := fmt.Sprintf("%s/cgi-bin/message/template/send?access_token=%s", cfg.Base(), url.QueryEscape(accessToken))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(encoded)))
	if err != nil {
		return fmt.Errorf("wechat: build send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("wechat: send request: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("wechat: read send response: %w", err)
	}

	var parsed wechatSendResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("wechat: decode send response: %w", err)
	}
	if parsed.ErrCode != 0 {
		return fmt.Errorf("wechat: send failed: %d %s", parsed.ErrCode, parsed.ErrMsg)
	}
	return nil
}
