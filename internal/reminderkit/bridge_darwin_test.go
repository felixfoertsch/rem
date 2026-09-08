//go:build darwin && cgo

package reminderkit

import (
	"sync"
	"testing"
)

// Real native calls, but never a store, user-data read, or permission request.
func TestNativeDiagnosticsWithoutAccess(t *testing.T) {
	client := New()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := client.Diagnostics()
			if err != nil { t.Error(err); return }
			if result["permission_requested"] != false { t.Errorf("unexpected permission request: %v", result) }
			if _, ok := result["reminders_authorization_status"]; !ok { t.Error("missing authorization status") }
		}()
	}
	wg.Wait()
}

func TestNativeMarshalErrorIsReturned(t *testing.T) {
	var result any
	if err := nativeCall(make(chan int), &result); err == nil { t.Fatal("unsupported request should fail before native code") }
}
