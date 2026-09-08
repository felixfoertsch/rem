//go:build darwin && cgo

package reminderkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework EventKit -framework Foundation
#include <stdlib.h>
char *rem_collaboration_call(const char *request);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"
)

// One native operation at a time; the Objective-C side creates a fresh store
// for every request so no cached backing object survives a save.
var nativeMu sync.Mutex

func nativeCall(request any, out any) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	input := C.CString(string(data))
	defer C.free(unsafe.Pointer(input))
	nativeMu.Lock()
	defer nativeMu.Unlock()
	result := C.rem_collaboration_call(input)
	if result == nil {
		return fmt.Errorf("ReminderKit returned no response")
	}
	defer C.free(unsafe.Pointer(result))
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal([]byte(C.GoString(result)), &envelope); err != nil {
		return fmt.Errorf("invalid ReminderKit response: %w", err)
	}
	if envelope.Error != "" {
		return fmt.Errorf("ReminderKit: %s", envelope.Error)
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("ReminderKit response has no result")
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("invalid ReminderKit result: %w", err)
	}
	return nil
}
