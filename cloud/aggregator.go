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

func (client *Client) Aggregators(ctx context.Context, projectID string, last uint64) ([]cloud.Aggregator, error) {
	u := cloneURL(client.BaseURL)
	u.Path = "/v1/projects/" + url.PathEscape(projectID) + "/aggregators"
	u.RawQuery = url.Values{
		"last": []string{strconv.FormatUint(last, 10)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create aggregators http request: %w", err)
	}

	req.Header.Set("User-Agent", "calyptia-cloud-cli")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch aggregators: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode aggregators error response: %w", err)
		}

		return nil, e
	}

	var aa []cloud.Aggregator
	err = json.NewDecoder(resp.Body).Decode(&aa)
	if err != nil {
		return nil, fmt.Errorf("could not json decode aggregators response: %w", err)
	}

	return aa, nil
}
