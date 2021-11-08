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

func (client *Client) PipelinePorts(ctx context.Context, pipelineID string, last uint64) ([]cloud.PipelinePort, error) {
	u := cloneURL(client.BaseURL)
	u.Path = "/v1/aggregator_pipelines/" + url.PathEscape(pipelineID) + "/ports"
	u.RawQuery = url.Values{
		"last": []string{strconv.FormatUint(last, 10)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create pipeline ports http request: %w", err)
	}

	req.Header.Set("User-Agent", "calyptia-cloud-cli")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch pipeline ports: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return nil, fmt.Errorf("could not json decode pipeline ports error response: %w", err)
		}

		return nil, e
	}

	var pp []cloud.PipelinePort
	err = json.NewDecoder(resp.Body).Decode(&pp)
	if err != nil {
		return nil, fmt.Errorf("could not json decode pipeline ports response: %w", err)
	}

	return pp, nil
}
