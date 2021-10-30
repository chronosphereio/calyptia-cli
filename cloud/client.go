package cloud

import (
	"net/http"
	"net/url"
)

type Client struct {
	HTTPClient  *http.Client
	BaseURL     *url.URL
	AccessToken string
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := new(url.URL)
	*u2 = *u
	if u.User != nil {
		u2.User = new(url.Userinfo)
		*u2.User = *u.User
	}
	return u2
}
