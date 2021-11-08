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

func (client *Client) AggregatorPipelines(ctx context.Context, aggregatorID string, last uint64) ([]cloud.AggregatorPipeline, error) {
	u := cloneURL(client.BaseURL)
	u.Path = "/v1/aggregators/" + url.PathEscape(aggregatorID) + "/pipelines"
	u.RawQuery = url.Values{
		"last": []string{strconv.FormatUint(last, 10)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create aggregator pipelines http request: %w", err)
	}

	req.Header.Set("User-Agent", "calyptia-cloud-cli")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch aggregator pipelines: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode aggregator pipelines error response: %w", err)
		}

		return nil, e
	}

	var pp []cloud.AggregatorPipeline
	err = json.NewDecoder(resp.Body).Decode(&pp)
	if err != nil {
		return nil, fmt.Errorf("could not json decode aggregator pipelines response: %w", err)
	}

	return pp, nil
}
