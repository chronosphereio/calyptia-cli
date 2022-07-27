package aws

import (
	"fmt"
	"strings"
)

var (
	ErrSubnetNotFound         = fmt.Errorf("subnet not found")
	ErrInstanceStatusNotFound = fmt.Errorf("instance status not found")
	ErrKeyPairNotFound        = fmt.Errorf("key pair not found")
)

func errorIsAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), ".Duplicate")
}

func errorIsNotFound(err error) bool {
	return strings.Contains(err.Error(), ".NotFound")
}
