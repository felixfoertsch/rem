package reminderkit

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/BRO3886/rem/internal/reminder"
)

func TestResolveParticipant(t *testing.T) {
	people := []reminder.Participant{
		{ID: "A", Name: "Alex", Address: "alex@example.com", IsMe: true},
		{ID: "B", Name: "Alex", Address: "other@example.com"},
		{ID: "C", Name: "Zoë", Address: "MAILTO:zoe@example.com"},
	}
	for _, tc := range []struct{ query, want string }{
		{"a", "A"}, {" me ", "A"}, {"ALEX@EXAMPLE.COM", "A"}, {"mailto:OTHER@example.com", "B"}, {"zoë", "C"}, {"MAILTO:ZOE@example.com", "C"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			got, err := ResolveParticipant(people, tc.query)
			if err != nil || got.ID != tc.want {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
	for _, q := range []string{"Alex", "al", "", "missing@example.com"} {
		if _, err := ResolveParticipant(people, q); err == nil {
			t.Errorf("accepted ambiguous/missing query %q", q)
		}
	}
	if _, err := ResolveParticipant([]reminder.Participant{{ID: "A", IsMe: true}, {ID: "B", IsMe: true}}, "me"); err == nil {
		t.Fatal("ambiguous native identity accepted")
	}
	if _, err := ResolveParticipant([]reminder.Participant{{ID: "A", Name: "me"}}, "me"); err == nil {
		t.Fatal("name must not impersonate native me identity")
	}
	if _, err := ResolveParticipant([]reminder.Participant{{Name: "No ID"}}, "No ID"); err == nil {
		t.Fatal("missing ID accepted")
	}
}

func fill(out any, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func TestAssignValidatesAndVerifies(t *testing.T) {
	for _, clear := range []bool{false, true} {
		calls := 0
		client := &Client{call: func(request, out any) error {
			calls++
			r := request.(map[string]any)
			if calls == 1 {
				if r["op"] != "roster" {
					t.Fatalf("first operation = %v", r)
				}
				return fill(out, map[string]any{"list_id": "L", "people": []reminder.Participant{{ID: "P", Name: "Pat"}}})
			}
			if r["op"] != "assign" || r["list_id"] != "L" || r["clear"] != clear {
				t.Fatalf("unexpected write %v", r)
			}
			result := reminder.Collaboration{AssignmentAvailable: true}
			if !clear {
				if r["participant_id"] != "P" {
					t.Fatal("not a resolved ID")
				}
				result.AssignedTo = &reminder.Participant{ID: "P"}
			}
			return fill(out, result)
		}}
		query := "Pat"
		if clear {
			query = ""
		}
		if _, err := client.Assign("R", query, clear); err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("calls %d", calls)
		}
	}
}

func TestAssignFailureDoesNotReportSuccess(t *testing.T) {
	for _, mode := range []string{"roster-error", "no-list-id", "unknown-person", "save-error", "unavailable", "wrong-assignee", "uncleared"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			client := &Client{call: func(request, out any) error {
				r := request.(map[string]any)
				if r["op"] == "roster" {
					if mode == "roster-error" {
						return errors.New("roster failed")
					}
					list := "L"
					if mode == "no-list-id" {
						list = ""
					}
					people := []reminder.Participant{{ID: "P", Name: "Pat"}}
					if mode == "unknown-person" {
						people = nil
					}
					return fill(out, map[string]any{"list_id": list, "people": people})
				}
				writes++
				if mode == "save-error" {
					return errors.New("save failed")
				}
				if mode == "unavailable" {
					return fill(out, reminder.Collaboration{})
				}
				return fill(out, reminder.Collaboration{AssignmentAvailable: true, AssignedTo: &reminder.Participant{ID: "WRONG"}})
			}}
			q := "Pat"
			clear := mode == "uncleared"
			if clear {
				q = ""
			}
			if _, err := client.Assign("R", q, clear); err == nil {
				t.Fatal("failure reported success")
			}
			if (mode == "roster-error" || mode == "unknown-person" || mode == "no-list-id") && writes != 0 {
				t.Fatal("unsafe write after failed resolution")
			}
		})
	}
}

func TestInvalidRequestsNeverCrossBridge(t *testing.T) {
	client := &Client{call: func(any, any) error { t.Fatal("invalid request called native bridge"); return nil }}
	for _, v := range []struct {
		id, q string
		clear bool
	}{{"", "Pat", false}, {"R", "", false}, {"R", "Pat", true}, {" ", "", true}} {
		if _, err := client.Assign(v.id, v.q, v.clear); err == nil {
			t.Fatalf("accepted %v", v)
		}
	}
	if data, err := client.Metadata(nil); err != nil || len(data) != 0 {
		t.Fatalf("empty metadata %v %v", data, err)
	}
	if err := client.ChangeSection("section-delete", "L", "S", ""); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("delete unexpectedly supported: %v", err)
	}
}
