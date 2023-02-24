package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/hako/durafmt"
	"github.com/spf13/cobra"

	"github.com/calyptia/api/client"
	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/localdata"
)

type Config struct {
	Ctx          context.Context
	BaseURL      string
	Cloud        *client.Client
	ProjectToken string
	ProjectID    string
	LocalData    *localdata.Keyring
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

func CompleteOutputFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml", "go-template"}, cobra.ShellCompDirectiveNoFileComp
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

var reUUID4 = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

func Ptr[T any](p T) *T { return &p }

func ValidUUID(s string) bool {
	return reUUID4.MatchString(s)
}
