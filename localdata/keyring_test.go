package localdata

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

func TestKeyring(t *testing.T) {
	t.Run("Save", func(t *testing.T) {
		kr := New("save-test", ".backup-file")
		err := kr.Save("key", "data")
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		get, err := kr.Get("key")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if get != "data" {
			t.Errorf("Get() got = %v, want %v", get, "data")
		}
		err = kr.Delete("key")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})
	t.Run("Get", func(t *testing.T) {
		t.Run("NotFound", func(t *testing.T) {
			kr := New("get-test", ".backup-file")
			get, err := kr.Get("key")
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected ErrNotFound, got %v", err)
			}
			if get != "" {
				t.Errorf("Get() got = %v, want %v", get, "")
			}
		})
		t.Run("Found", func(t *testing.T) {
			kr := New("get-test", ".backup-file")
			err := kr.Save("key", "data")
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			get, err := kr.Get("key")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if get != "data" {
				t.Errorf("Get() got = %v, want %v", get, "data")
			}
		})
	})
	t.Run("Delete", func(t *testing.T) {
		t.Run("NotFound", func(t *testing.T) {
			kr := New("delete-test", ".backup-file")
			err := kr.Delete("key")
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected ErrNotFound, got %v", err)
			}
		})
		t.Run("Success", func(t *testing.T) {
			kr := New("delete-test", ".backup-file")
			err := kr.Save("key", "data")
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			err = kr.Delete("key")
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			get, err := kr.Get("key")
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected ErrNotFound, got %v", err)
			}
			if get != "" {
				t.Errorf("Get() got = %v, want %v", get, "")
			}
		})
	})
}
