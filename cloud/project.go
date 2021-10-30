package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/calyptia/cloud"
)

const userAgent = "calyptia-cloud-api"

func (client *Client) Projects(ctx context.Context, last uint64) ([]cloud.Project, error) {
	u := cloneURL(client.BaseURL)
	u.Path = "/v1/projects"
	u.RawQuery = url.Values{
		"last": []string{strconv.FormatUint(last, 10)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create projects http request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	if client.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+client.AccessToken)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch projects: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode projects error response: %w", err)
		}

		return nil, e
	}

	var pp []cloud.Project
	err = json.NewDecoder(resp.Body).Decode(&pp)
	if err != nil {
		return nil, fmt.Errorf("could not json decode projects response: %w", err)
	}

	return pp, nil
}
