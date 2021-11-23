package main

import (
	"time"

	"github.com/hako/durafmt"
)

func fmtAgo(t time.Time) string {
	return durafmt.ParseShort(time.Since(t)).LimitFirstN(1).String()
}
