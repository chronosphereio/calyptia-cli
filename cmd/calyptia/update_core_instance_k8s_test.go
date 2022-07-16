package main

import "testing"

func TestVerifyCoreVersion(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		if err := VerifyCoreVersion("1.0.0", []string{"1.0.0", "1.0.1"}); err != nil {
			t.Error("expected no error, got", err)
		}
	})

	t.Run("expect error on invalid version", func(t *testing.T) {
		err := VerifyCoreVersion("vInvalid", []string{"1.0.0", "1.0.1"})
		if err == nil {
			t.Error("expected error")
		}
		if err.Error() != "version vInvalid is not available" {
			t.Errorf("expected error to be 'core version 1.0.0 is not supported'")
		}
	})
}
