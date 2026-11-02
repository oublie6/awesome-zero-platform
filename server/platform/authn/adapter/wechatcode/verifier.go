package wechatcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidCode = errors.New("invalid WeChat login code")

type Verifier struct {
	appID    string
	secret   string
	client   *http.Client
	endpoint string
}

func New(appID, secret string) (*Verifier, error) {
	if len(appID) < 4 || len(appID) > 128 || len(secret) < 16 {
		return nil, fmt.Errorf("invalid WeChat exchange configuration")
	}
	return &Verifier{appID: appID, secret: secret, endpoint: "https://api.weixin.qq.com/sns/jscode2session",
		client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// Exchange calls WeChat over HTTPS and returns only the verified opaque OpenID.
// Session keys, client codes and application secrets are never logged or
// propagated to business modules.
func (v *Verifier) Exchange(ctx context.Context, code string) (string, error) {
	if v == nil || v.client == nil || len(code) < 4 || len(code) > 128 ||
		code != strings.TrimSpace(code) || strings.ContainsAny(code, " \r\n\t") {
		return "", ErrInvalidCode
	}
	query := url.Values{"appid": {v.appID}, "secret": {v.secret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("prepare WeChat exchange: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("WeChat identity exchange failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ErrInvalidCode
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2049))
	if err != nil || len(body) > 2048 {
		return "", ErrInvalidCode
	}
	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.ErrCode != 0 ||
		len(result.OpenID) < 4 || len(result.OpenID) > 256 ||
		result.OpenID != strings.TrimSpace(result.OpenID) {
		return "", ErrInvalidCode
	}
	return result.OpenID, nil
}
