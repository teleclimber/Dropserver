//go:build !linux

package sandbox

import "errors"

func NewBwrapJsonStatus(basePath string) (BwrapStatusJsonI, error) {
	return nil, errors.New("NewBwrapJsonStatus should not be called unless compiled for Linux")
}
