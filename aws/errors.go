package aws

import "strings"

func errorIsAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), ".Duplicate")
}

func errorIsNotFound(err error) bool {
	return strings.Contains(err.Error(), ".NotFound")
}
