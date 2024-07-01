// Package uuid provides UUID v4 utilities.
package uuid

import "regexp"

var re = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

// Valid returns whether the given string is a valid UUID v4.
func Valid(s string) bool {
	return re.MatchString(s)
}
