package aws

import (
	"github.com/pkg/errors"
	"strings"
)

var (
	ErrSubnetNotFound = errors.New("subnet not found")
)

func errorIsAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), ".Duplicate")
}

func errorIsNotFound(err error) bool {
	return strings.Contains(err.Error(), ".NotFound")
}
