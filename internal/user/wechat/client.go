package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	AppID     string
	AppSecret string
}

type Client struct {
	config     *Config
	httpClient *http.Client
}

type Code2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func NewClient(cfg *Config) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Code2Session(ctx context.Context, code string) (*Code2SessionResponse, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		c.config.AppID,
		c.config.AppSecret,
		code,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result Code2SessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

func (c *Client) GetOpenID(ctx context.Context, code string) (string, error) {
	resp, err := c.Code2Session(ctx, code)
	if err != nil {
		return "", err
	}

	return resp.OpenID, nil
}
