package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func testConfig(mock *ClientMock) *config {
	if mock == nil {
		mock = &ClientMock{}
	}
	return &config{
		ctx:          context.Background(),
		cloud:        mock,
		projectID:    "",
		projectToken: "",
	}
}

func wantEq(t *testing.T, want, got interface{}) {
	t.Helper()

	if target, ok := want.(error); ok {
		if err, ok := got.(error); ok {
			if !errors.Is(err, target) {
				t.Fatalf("want %v, got %v", target, err)
			}
			return
		}
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %+v, got %+v", want, got)
	}
}

func wantNoEq(t *testing.T, want, got interface{}) {
	t.Helper()

	if reflect.DeepEqual(want, got) {
		t.Fatalf("want %+v not equal to %+v", want, got)
	}
}

func wantErrMsg(t *testing.T, want string, got error) {
	if got == nil || !strings.Contains(got.Error(), want) {
		t.Fatalf("want error message %q, got %v", want, got)
	}
}
