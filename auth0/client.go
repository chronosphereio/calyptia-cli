package auth0

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

// IsAuthorizationPendingError: You will see this error while waiting for the user to take action. Continue polling using the suggested interval retrieved in the previous step of this tutorial.
func IsAuthorizationPendingError(err error) bool {
	var e Error
	return errors.As(err, &e) && e.Msg == "authorization_pending"
}

// IsSlowDownError: You are polling too fast. Slow down and use the suggested interval retrieved in the previous step of this tutorial. To avoid receiving this error due to network latency, you should start counting each interval after receipt of the last polling request's response.
func IsSlowDownError(err error) bool {
	var e Error
	return errors.As(err, &e) && e.Msg == "slow_down"
}

// IsExpiredTokenError: The user has not authorized the device quickly enough, so the `device_code` has expired. Your application should notify the user that the flow has expired and prompt them to reinitiate the flow.
// The `expired_token` error will be returned exactly once; after that, the dreaded `invalid_grant` will be returned. Your device **must** stop polling.
func IsExpiredTokenError(err error) bool {
	var e Error
	return errors.As(err, &e) && e.Msg == "expired_token"
}

// IsAccessDeniedError: Finally, if access is denied, you will receive this.
// This can occur for a variety of reasons, including:
//   - the user refused to authorize the device
//   - the authorization server denied the transaction
//   - a configured rule denied access.
func IsAccessDeniedError(err error) bool {
	var e Error
	return errors.As(err, &e) && e.Msg == "access_denied"
}

type Client struct {
	HTTPClient *http.Client
	Domain     string
	ClientID   string
	Audience   string
}

type DeviceCode struct {
	// DeviceCode is the unique code for the device. When the user goes to the verification_uri in their browser-based device, this code will be bound to their session.
	DeviceCode string `json:"device_code"`
	// UserCode contains the code that should be input at the `verification_uri` to authorize the device.
	UserCode string `json:"user_code"`
	// VerificationURI contains the URL the user should visit to authorize the device.
	VerificationURI string `json:"verification_uri"`
	// VerificationURIComplete contains the complete URL the user should visit to authorize the device.
	// This allows your app to embed the `user_code` in the URL, if you so choose.
	VerificationURIComplete string `json:"verification_uri_complete"`
	// ExpiresIn indicates the lifetime (in seconds) of the `device_code` and `user_code`.
	ExpiresIn int64 `json:"expires_in"`
	// Interval indicates the interval (in seconds) at which the app should poll the token URL to request a token.
	Interval int64 `json:"interval"`
}

func (client *Client) DeviceCode(ctx context.Context) (DeviceCode, error) {
	var dc DeviceCode

	body := url.Values{}
	body.Set("client_id", client.ClientID)
	body.Set("scope", "user email openid offline_access")
	body.Set("audience", client.Audience)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+client.Domain+"/oauth/device/code", strings.NewReader(body.Encode()))
	if err != nil {
		return dc, fmt.Errorf("could not create device code http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return dc, fmt.Errorf("could not http fetch device code: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			return dc, fmt.Errorf("could not read device code http error response: status_code=%d: %w", resp.StatusCode, err)
		}

		var e Error
		err = json.Unmarshal(got, &e)
		if err != nil {
			return dc, fmt.Errorf("could not json decode device code error http response: status_code=%d body=%s: %w", resp.StatusCode, string(got), err)
		}

		return dc, e
	}

	err = json.NewDecoder(resp.Body).Decode(&dc)
	if err != nil {
		return dc, fmt.Errorf("could not json decode device code http response: %w", err)
	}

	return dc, nil
}

type Error struct {
	Msg  string `json:"error"`
	Desc string `json:"description"`
}

func (e Error) Error() string {
	if e.Desc == "" {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", e.Msg, e.Desc)
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`

	// ExpiresAt does not come from the API. It is added using `ExpiresIn`.
	ExpiresAt time.Time `json:"expires_at"`
}

func (client *Client) AccessToken(ctx context.Context, deviceCode string) (AccessToken, error) {
	var tok AccessToken

	body := url.Values{}
	body.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	body.Set("device_code", deviceCode)
	body.Set("client_id", client.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+client.Domain+"/oauth/token", strings.NewReader(body.Encode()))
	if err != nil {
		return tok, fmt.Errorf("could not create access token http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	now := time.Now()
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return tok, fmt.Errorf("could not http fetch access token: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			return tok, fmt.Errorf("could not read access token http error response: status_code=%d: %w", resp.StatusCode, err)
		}

		var e Error
		err = json.Unmarshal(got, &e)
		if err != nil {
			return tok, fmt.Errorf("could not json decode access token error http response: status_code=%d body=%s: %w", resp.StatusCode, string(got), err)
		}

		return tok, e
	}

	err = json.NewDecoder(resp.Body).Decode(&tok)
	if err != nil {
		return tok, fmt.Errorf("could not json decode access token http response: %w", err)
	}

	tok.ExpiresAt = now.Add(time.Second * time.Duration(tok.ExpiresIn))

	return tok, nil
}

type RefreshToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`

	// ExpiresAt does not come from the API. It is added using `ExpiresIn`.
	ExpiresAt time.Time `json:"expires_at"`
}

func (client *Client) RefreshToken(ctx context.Context, refreshToken string) (RefreshToken, error) {
	var tok RefreshToken

	body := url.Values{}
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", refreshToken)
	body.Set("client_id", client.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+client.Domain+"/oauth/token", strings.NewReader(body.Encode()))
	if err != nil {
		return tok, fmt.Errorf("could not create refresh token http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	now := time.Now()
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return tok, fmt.Errorf("could not http fetch refresh token: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			return tok, fmt.Errorf("could not read refresh token http error response: status_code=%d: %w", resp.StatusCode, err)
		}

		var e Error
		err = json.Unmarshal(got, &e)
		if err != nil {
			return tok, fmt.Errorf("could not json decode refresh token error http response: status_code=%d body=%s: %w", resp.StatusCode, string(got), err)
		}

		return tok, e
	}

	err = json.NewDecoder(resp.Body).Decode(&tok)
	if err != nil {
		return tok, fmt.Errorf("could not json decode refresh token http response: %w", err)
	}

	tok.ExpiresAt = now.Add(time.Second * time.Duration(tok.ExpiresIn))

	return tok, nil
}
