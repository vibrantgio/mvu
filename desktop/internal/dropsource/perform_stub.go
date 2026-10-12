//go:build !darwin

package dropsource

import "errors"

func begin(string, *Promise) error {
	return errors.New("dropsource: a drag can only be driven on macOS")
}

func send(call, float64, float64) (bool, error) {
	return false, errors.New("dropsource: a drag can only be driven on macOS")
}
