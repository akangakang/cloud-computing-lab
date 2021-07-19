package fsclient

import (
	"fmt"
	fsrpc "github.com/fourstring/sheetfs/protocol"
)

type CancelledError struct {
}

func (c *CancelledError) Error() string {
	return "Operation was cancelled!"
}

type UnexpectedStatusError struct {
	status fsrpc.Status
}

func NewUnexpectedStatusError(status fsrpc.Status) *UnexpectedStatusError {
	return &UnexpectedStatusError{status: status}
}

func (u *UnexpectedStatusError) Error() string {
	return fmt.Sprintf("Operation returned unexpected status: %s", u.status)
}

func (u *UnexpectedStatusError) Status() fsrpc.Status {
	return u.status
}
