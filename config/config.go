package config

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hako/durafmt"

	"github.com/calyptia/api/client"
	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/localdata"
)

type Config struct {
	BaseURL      string
	Cloud        *client.Client
	ProjectToken string
	ProjectID    string
	LocalData    *localdata.Keyring
	Completer    *completer.Completer
}

func AgentStatus(lastMetricsAddedAt *time.Time, start time.Duration) string {
	var status string
	if lastMetricsAddedAt == nil || lastMetricsAddedAt.IsZero() {
		status = "inactive"
	} else if lastMetricsAddedAt.Before(time.Now().Add(start)) {
		status = fmt.Sprintf("inactive for %s", durafmt.ParseShort(time.Since(*lastMetricsAddedAt)).LimitFirstN(1))
	} else {
		status = "active"
	}
	return status
}

func PairsName(pp cloud.Pairs) string {
	if v, ok := pp.Get("Name"); ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func UniqueSlice[S ~[]E, E comparable](s S) S {
	m := map[E]struct{}{}

	var out S
	for _, item := range s {
		if _, ok := m[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}

func Env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func ReadFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read contents: %w", err)
	}

	return b, nil
}
