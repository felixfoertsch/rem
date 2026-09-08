//go:build !darwin || !cgo

package reminderkit

import "fmt"

func nativeCall(any, any) error {
	return fmt.Errorf("shared reminders and sections require macOS with cgo enabled")
}
