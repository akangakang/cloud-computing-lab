package common_journal

import "fmt"

type InvalidVersionError struct {
	version int64
}

func NewInvalidVersionError(version int64) *InvalidVersionError {
	return &InvalidVersionError{version: version}
}

func (i *InvalidVersionError) Error() string {
	return fmt.Sprintf("version %d is invalid!", i.version)
}

type NoMoreMessageError struct {
}

func (n *NoMoreMessageError) Error() string {
	return fmt.Sprintf("All messages has been consumed!")
}
