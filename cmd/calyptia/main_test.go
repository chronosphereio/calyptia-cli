package main

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// func configWithMock(mock *ClientMock) *utils.Config {
// 	if mock == nil {
// 		mock = &ClientMock{}
// 	}
// 	return &utils.Config{
// 		Ctx:          context.Background(),
// 		Cloud:        mock,
// 		ProjectID:    "",
// 		ProjectToken: "",
// 	}
// }

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
	t.Helper()

	if got == nil || !strings.Contains(got.Error(), want) {
		t.Fatalf("want error message %q, got %v", want, got)
	}
}

func setupFile(t *testing.T, name string, contents []byte) *os.File {
	t.Helper()

	dir := t.TempDir()
	f, err := os.CreateTemp(dir, name)
	wantEq(t, nil, err)

	t.Cleanup(func() {
		f.Close()
	})

	if contents != nil {
		_, err = f.Write(contents)
		wantEq(t, nil, err)
	}

	return f
}
