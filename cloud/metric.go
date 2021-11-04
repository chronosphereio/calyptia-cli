package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/calyptia/cloud"
)

func (client *Client) Metrics(ctx context.Context, projectID string, start, interval time.Duration) (cloud.ProjectMetrics, error) {
	var pm cloud.ProjectMetrics

	u := cloneURL(client.BaseURL)
	u.Path = "/v1/projects/" + url.PathEscape(projectID) + "/metrics"
	u.RawQuery = url.Values{
		"start":    []string{start.String()},
		"interval": []string{interval.String()},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return pm, fmt.Errorf("could not create metrics http request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return pm, fmt.Errorf("could not fetch metrics: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return pm, fmt.Errorf("could not json decode metrics error response: %w", err)
		}

		return pm, e
	}

	err = json.NewDecoder(resp.Body).Decode(&pm)
	if err != nil {
		return pm, fmt.Errorf("could not json decode metricss response: %w", err)
	}

	return pm, nil
}
