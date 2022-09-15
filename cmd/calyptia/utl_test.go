package main

import (
	"bytes"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func Test_applyGoTemplate(t *testing.T) {
	t.Run("text_inline", func(t *testing.T) {
		var buff bytes.Buffer
		err := applyGoTemplate(&buff, "go-template='{{range .}}{{.}}{{end}}'", "", []string{"foo", "bar"})
		assert.NoError(t, err)

		got := buff.String()
		assert.Equal(t, "foobar\n", got)
	})

	t.Run("text_with_separate_template", func(t *testing.T) {
		var buff bytes.Buffer
		err := applyGoTemplate(&buff, "go-template", "{{range .}}{{.}}{{end}}", []string{"foo", "bar"})
		assert.NoError(t, err)

		got := buff.String()
		assert.Equal(t, "foobar\n", got)
	})
}
