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

func (client *Client) Agents(ctx context.Context, projectID string, last uint64) ([]cloud.Agent, error) {
	u := cloneURL(client.BaseURL)
	u.Path = "/v1/projects/" + url.PathEscape(projectID) + "/agents"
	u.RawQuery = url.Values{
		"last": []string{strconv.FormatUint(last, 10)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create agents http request: %w", err)
	}

	req.Header.Set("User-Agent", "calyptia-cloud-cli")
	if client.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+client.AccessToken)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch agents: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode agents error response: %w", err)
		}

		return nil, e
	}

	var aa []cloud.Agent
	err = json.NewDecoder(resp.Body).Decode(&aa)
	if err != nil {
		return nil, fmt.Errorf("could not json decode agents response: %w", err)
	}

	return aa, nil
}
