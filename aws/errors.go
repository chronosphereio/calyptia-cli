package aws

import (
	"fmt"
	"strings"
)

var (
	ErrSubnetNotFound        = fmt.Errorf("subnet not found")
	ErrElasticIPAddressInUse = fmt.Errorf("elastic ip address is already in use")
)

func errorIsAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), ".Duplicate")
}

func errorIsNotFound(err error) bool {
	return strings.Contains(err.Error(), ".NotFound")
}
