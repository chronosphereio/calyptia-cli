package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	githubBaseURL = "https://api.github.com/repos/calyptia/"
)

func (c *GithubClient) GetTags(repo string) (*Latest, *ErrorResponse, error) {
	i := &Latest{}

	path := fmt.Sprintf("%s/releases/latest", repo)
	if resp, err := c.DoRequest("GET", path, nil, i); err == nil {
		return i, resp, err
	} else {
		return nil, resp, err
	}
}

func (c *GithubClient) GetLatestTag(repo string) (string, error) {
	i, _, err := c.GetTags(repo)
	if err != nil {
		return "", err
	}

	return i.TagName, nil
}

func NewGithubClient(apiKey string, httpClient *http.Client) *GithubClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	url, _ := url.Parse(githubBaseURL)

	client := &GithubClient{
		apiKey:  apiKey,
		baseURL: url,
		client:  httpClient,
	}

	return client
}

type GithubClient struct {
	client  *http.Client
	apiKey  string
	baseURL *url.URL
}

type ErrorResponse struct {
	StatusCode int
	Errors     string
}

func (c *GithubClient) DoRequest(method, path string, body, v interface{}) (*ErrorResponse, error) {
	req, err := c.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	return c.Do(req, v)
}

func (c *GithubClient) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	// relative path to append to the endpoint url, no leading slash please
	if path[0] == '/' {
		path = path[1:]
	}
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(rel)
	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, u.String(), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, u.String(), nil)
	}
	if err != nil {
		return nil, err
	}

	req.Close = true

	req.Header.Add("MC-Api-Key", c.apiKey)
	return req, nil
}

func (c *GithubClient) Do(req *http.Request, v interface{}) (*ErrorResponse, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		o, _ := io.ReadAll(resp.Body)
		errResp := &ErrorResponse{
			StatusCode: resp.StatusCode,
			Errors:     string(o),
		}

		return errResp, fmt.Errorf("%s returned %d", req.URL, resp.StatusCode)
	}

	o, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(o, v)
	if err != nil {
		return nil, err
	}

	return nil, err
}

type Tag struct {
	Ref    string `json:"ref,omitempty"`
	NodeID string `json:"node_id,omitempty"`
	URL    string `json:"url,omitempty"`
	Object Obj    `json:"object,omitempty"`
}

type Obj struct {
	SHA   string `json:"sha,omitempty"`
	TType string `json:"type,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Latest struct {
	TagName string `json:"tag_name"`
}
