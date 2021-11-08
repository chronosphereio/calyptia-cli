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
	var metrics cloud.ProjectMetrics

	u := cloneURL(client.BaseURL)
	u.Path = "/v1/projects/" + url.PathEscape(projectID) + "/metrics"
	u.RawQuery = url.Values{
		"start":    []string{start.String()},
		"interval": []string{interval.String()},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return metrics, fmt.Errorf("could not create metrics http request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return metrics, fmt.Errorf("could not fetch metrics: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return metrics, fmt.Errorf("could not json decode metrics error response: %w", err)
		}

		return metrics, e
	}

	err = json.NewDecoder(resp.Body).Decode(&metrics)
	if err != nil {
		return metrics, fmt.Errorf("could not json decode metrics response: %w", err)
	}

	return metrics, nil
}

func (client *Client) AgentMetrics(ctx context.Context, agentID string, start, interval time.Duration) (cloud.AgentMetrics, error) {
	var metrics cloud.AgentMetrics

	u := cloneURL(client.BaseURL)
	u.Path = "/v1/agents/" + url.PathEscape(agentID) + "/metrics"
	u.RawQuery = url.Values{
		"start":    []string{start.String()},
		"interval": []string{interval.String()},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return metrics, fmt.Errorf("could not create agent metrics http request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return metrics, fmt.Errorf("could not fetch agent metrics: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &cloud.Error{}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return metrics, fmt.Errorf("could not json decode agent metrics error response: %w", err)
		}

		return metrics, e
	}

	err = json.NewDecoder(resp.Body).Decode(&metrics)
	if err != nil {
		return metrics, fmt.Errorf("could not json decode agent metrics response: %w", err)
	}

	return metrics, nil
}
