package ns

import (
	"fmt"
)

// NefError - nef error format
type NefError struct {
	Err  error
	Code string
}

func (e *NefError) Error() string {
	return fmt.Sprintf("%s [code: %s]", e.Err, e.Code)
}

// GetNefErrorCode - treats an error as NefError and returns its code in case of success
func GetNefErrorCode(err error) string {
	if nefErr, ok := err.(*NefError); ok {
		return nefErr.Code
	}
	return ""
}

// IsAlreadyExistNefError - treats an error as NefError and returns true if its code is "EEXIST"
func IsAlreadyExistNefError(err error) bool {
	return GetNefErrorCode(err) == "EEXIST"
}

// IsNotExistNefError - treats an error as NefError and returns true if its code is "ENOENT"
func IsNotExistNefError(err error) bool {
	return GetNefErrorCode(err) == "ENOENT"
}

// IsAuthNefError - treats an error as NefError and returns true if its code is "EAUTH"
func IsAuthNefError(err error) bool {
	return GetNefErrorCode(err) == "EAUTH" //TODO use constants
}
