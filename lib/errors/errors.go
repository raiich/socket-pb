package errors

import (
	"errors"
	"fmt"
)

func Newf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

func Wrapf(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%w: %s", err, fmt.Sprintf(format, args...))
}

func Is(actual, expected error) bool {
	return errors.Is(actual, expected)
}
