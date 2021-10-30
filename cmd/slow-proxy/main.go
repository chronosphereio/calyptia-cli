package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func main() {
	proxy := slowProxy(&url.URL{
		Scheme: "https",
		Host:   "cloud-api-dev.calyptia.com",
	}, time.Second*3)
	fmt.Println("strating slow proxy at http://localhost:5001")
	http.ListenAndServe(":5001", proxy)
}

func slowProxy(target *url.URL, delay time.Duration) *httputil.ReverseProxy {
	director := func(newr *http.Request) {
		target := cloneURL(target)
		newr.Host = target.Host
		newr.URL.Scheme = target.Scheme
		newr.URL.Host = target.Host

		// taken from httputil.NewSingleHostReverseProxy
		if _, ok := newr.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			newr.Header.Set("User-Agent", "")
		}

		time.Sleep(delay)
	}
	return &httputil.ReverseProxy{
		Director: director,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusBadGateway)
		},
	}
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
