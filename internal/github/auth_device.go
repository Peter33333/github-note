package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	deviceCodeEndpoint = "https://github.com/login/device/code"
	tokenEndpoint      = "https://github.com/login/oauth/access_token"
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

func requestDeviceCode(ctx context.Context, httpClient *http.Client, clientID string) (*deviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "repo read:user")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request device code failed: status=%d", resp.StatusCode)
	}

	result := &deviceCodeResponse{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, fmt.Errorf("decode device code response: %w", err)
	}
	return result, nil
}

func pollAccessToken(ctx context.Context, httpClient *http.Client, clientID string, deviceCode string, intervalSec int) (*oauth2.Token, error) {
	if intervalSec <= 0 {
		intervalSec = 5
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(intervalSec) * time.Second):
			form := url.Values{}
			form.Set("client_id", clientID)
			form.Set("device_code", deviceCode)
			form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
			if err != nil {
				return nil, fmt.Errorf("create token request: %w", err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("request token: %w", err)
			}

			result := &accessTokenResponse{}
			err = json.NewDecoder(resp.Body).Decode(result)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("decode token response: %w", err)
			}
			if result.Error == "authorization_pending" {
				continue
			}
			if result.Error == "slow_down" {
				intervalSec += 5
				continue
			}
			if result.Error != "" {
				if result.ErrorDesc != "" {
					return nil, fmt.Errorf("device flow error: %s (%s)", result.Error, result.ErrorDesc)
				}
				return nil, fmt.Errorf("device flow error: %s", result.Error)
			}
			if result.AccessToken == "" {
				return nil, fmt.Errorf("empty access token from device flow")
			}
			return &oauth2.Token{AccessToken: result.AccessToken, TokenType: result.TokenType}, nil
		}
	}
}
