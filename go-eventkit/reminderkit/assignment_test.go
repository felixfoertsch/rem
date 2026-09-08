package reminderkit

import (
	"errors"
	"testing"
)

func TestAssignmentBoundary(t *testing.T) {
	for _, mode := range []string{"assigned", "cleared", "save-error", "unavailable", "wrong", "uncleared"} {
		t.Run(mode, func(t *testing.T) {
			clear := mode == "cleared" || mode == "uncleared"
			participant := "P"
			if clear {
				participant = ""
			}
			c := &Client{call: func(request, out any) error {
				r := request.(map[string]any)
				if r["op"] != "assign" || r["id"] != "R" || r["list_id"] != "L" || r["participant_id"] != participant || r["clear"] != clear {
					t.Fatalf("bad transaction %v", r)
				}
				if mode == "save-error" {
					return errors.New("save failed")
				}
				result := Collaboration{AssignmentAvailable: mode != "unavailable"}
				if !clear || mode == "uncleared" {
					result.AssignedTo = &Participant{ID: "P"}
				}
				if mode == "wrong" {
					result.AssignedTo.ID = "OTHER"
				}
				return fill(out, result)
			}}
			_, err := c.SetAssignment("R", "L", participant)
			wantSuccess := mode == "assigned" || mode == "cleared"
			if (err == nil) != wantSuccess {
				t.Fatalf("success=%t error=%v", wantSuccess, err)
			}
		})
	}
	c := &Client{call: func(any, any) error { t.Fatal("invalid input crossed boundary"); return nil }}
	for _, ids := range [][3]string{{"", "L", "P"}, {"R", "", "P"}, {"R", "L", " "}} {
		if _, err := c.SetAssignment(ids[0], ids[1], ids[2]); err == nil {
			t.Fatal("invalid IDs accepted")
		}
	}
}

func TestRosterPinsNativeIdentity(t *testing.T) {
	for _, mode := range []string{"valid", "missing-list", "missing-reminder", "error"} {
		c := &Client{call: func(request, out any) error {
			if mode == "error" {
				return errors.New("denied")
			}
			r := Roster{ReminderID: "R", ListID: "L"}
			if mode == "missing-list" {
				r.ListID = ""
			}
			if mode == "missing-reminder" {
				r.ReminderID = ""
			}
			return fill(out, r)
		}}
		_, err := c.Roster("prefix")
		if (err == nil) != (mode == "valid") {
			t.Fatalf("%s: %v", mode, err)
		}
	}
}
