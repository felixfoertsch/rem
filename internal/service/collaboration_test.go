//go:build darwin

package service

import (
	"errors"
	"testing"

	"github.com/felixfoertsch/rem/internal/reminder"
)

type metadataFake struct {
	result map[string]*reminder.Collaboration
	err    error
	calls  int
	ids    []string
}

func (f *metadataFake) Metadata(ids []string) (map[string]*reminder.Collaboration, error) {
	f.calls++
	f.ids = ids
	return f.result, f.err
}

func TestCollaborationEnrichment(t *testing.T) {
	for _, mode := range []string{"assigned", "unassigned", "missing", "error"} {
		t.Run(mode, func(t *testing.T) {
			f := &metadataFake{result: map[string]*reminder.Collaboration{}}
			if mode == "error" {
				f.err = errors.New("unsupported")
			}
			if mode == "unassigned" || mode == "assigned" {
				f.result["A"] = &reminder.Collaboration{AssignmentAvailable: true}
			}
			if mode == "assigned" {
				f.result["A"].AssignedTo = &reminder.Participant{ID: "P", Name: "Pat"}
			}
			s := &ReminderService{collaboration: f}
			items := []*reminder.Reminder{{ID: "A", Name: "Keep title", Body: "Keep notes"}, {ID: "B"}}
			s.enrichCollaboration(items)
			if f.calls != 1 || len(f.ids) != 2 {
				t.Fatal("metadata must be batched")
			}
			if items[0].Name != "Keep title" || items[0].Body != "Keep notes" {
				t.Fatal("core fields changed")
			}
			if items[0].Collaboration == nil {
				t.Fatal("missing availability state")
			}
			if (mode == "error" || mode == "missing") && items[0].Collaboration.AssignmentAvailable {
				t.Fatal("unavailable misreported as unassigned")
			}
		})
	}
}
