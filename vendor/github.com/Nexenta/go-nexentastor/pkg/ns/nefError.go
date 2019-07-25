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

// IsNefError - checks if an error is an NefError
func IsNefError(err error) bool {
	_, ok := err.(*NefError)
	return ok
}

// GetNefErrorCode - treats an error as NefError and returns its code in case of success
func GetNefErrorCode(err error) string {
	if nefErr, ok := err.(*NefError); ok {
		return nefErr.Code
	}
	return ""
}

// IsAlreadyExistNefError treats an error as NefError and returns true if its code is "EEXIST"
func IsAlreadyExistNefError(err error) bool {
	return GetNefErrorCode(err) == "EEXIST"
}

// IsNotExistNefError treats an error as NefError and returns true if its code is "ENOENT"
func IsNotExistNefError(err error) bool {
	return GetNefErrorCode(err) == "ENOENT"
}

// IsBusyNefError treats an error as NefError and returns true if its code is "EBUSY"
// Example: filesystem cannot be deleted because it has snapshots
func IsBusyNefError(err error) bool {
	return GetNefErrorCode(err) == "EBUSY"
}

// IsAuthNefError treats an error as NefError and returns true if its code is "EAUTH"
func IsAuthNefError(err error) bool {
	return GetNefErrorCode(err) == "EAUTH" //TODO use constants
}

// IsBadArgNefError treats an error as NefError and returns true if its code is "EBADARG"
func IsBadArgNefError(err error) bool {
	return GetNefErrorCode(err) == "EBADARG"
}
