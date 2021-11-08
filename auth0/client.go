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

	"golang.org/x/oauth2"
)

// IsAuthorizationPendingError: You will see this error while waiting for the user to take action. Continue polling using the suggested interval retrieved in the previous step of this tutorial.
func IsAuthorizationPendingError(err error) bool {
	var e Error
	return errors.As(err, &e) && e.Msg == "authorization_pending"
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
	body.Set("scope", "user email offline_access")
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

type accessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (client *Client) AccessToken(ctx context.Context, deviceCode string) (*oauth2.Token, error) {
	body := url.Values{}
	body.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	body.Set("device_code", deviceCode)
	body.Set("client_id", client.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+client.Domain+"/oauth/token", strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("could not create access token http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	now := time.Now()
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not http fetch access token: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read access token http error response: status_code=%d: %w", resp.StatusCode, err)
		}

		var e Error
		err = json.Unmarshal(got, &e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode access token error http response: status_code=%d body=%s: %w", resp.StatusCode, string(got), err)
		}

		return nil, e
	}

	var tok accessToken
	err = json.NewDecoder(resp.Body).Decode(&tok)
	if err != nil {
		return nil, fmt.Errorf("could not json decode access token http response: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       now.Add(time.Second * time.Duration(tok.ExpiresIn)),
	}, nil
}

func (client *Client) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	tkr := &tokenRefresher{
		ctx:  ctx,
		conf: client,
	}
	if t != nil {
		tkr.refreshToken = t.RefreshToken
	}

	return oauth2.NewClient(ctx, oauth2.ReuseTokenSource(t, tkr))
}

type tokenRefresher struct {
	ctx          context.Context
	refreshToken string
	conf         *Client
}

type refreshToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (tf *tokenRefresher) Token() (*oauth2.Token, error) {
	body := url.Values{}
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", tf.refreshToken)
	body.Set("client_id", tf.conf.ClientID)
	req, err := http.NewRequestWithContext(tf.ctx, http.MethodPost, "https://"+tf.conf.Domain+"/oauth/token", strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("could not create refresh token http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	now := time.Now()
	resp, err := tf.conf.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not http fetch refresh token: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read refresh token http error response: status_code=%d: %w", resp.StatusCode, err)
		}

		var e Error
		err = json.Unmarshal(got, &e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode refresh token error http response: status_code=%d body=%s: %w", resp.StatusCode, string(got), err)
		}

		return nil, e
	}

	var tok refreshToken
	err = json.NewDecoder(resp.Body).Decode(&tok)
	if err != nil {
		return nil, fmt.Errorf("could not json decode refresh token http response: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tf.refreshToken,
		Expiry:       now.Add(time.Second * time.Duration(tok.ExpiresIn)),
	}, nil
}
