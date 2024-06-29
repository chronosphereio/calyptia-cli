// Package confirm provides utilities ask confirmation from stdin.
package confirm

import (
	"fmt"
	"io"
	"strings"
)

// Read whether the user confirmed the action by typing "y" or "yes".
func Read(r io.Reader) (bool, error) {
	var answer string
	_, err := fmt.Fscanln(r, &answer)
	if err != nil && err.Error() == "unexpected newline" {
		err = nil
	}

	if err != nil {
		return false, fmt.Errorf("could not to read answer: %v", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}
