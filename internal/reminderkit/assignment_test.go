package reminderkit

import (
	"errors"
	"testing"

	native "github.com/felixfoertsch/rem/go-eventkit/reminderkit"
)

type assignmentBackend struct {
	backend
	roster      *native.Roster
	err         error
	writes      int
	participant string
}

func (b *assignmentBackend) Roster(string) (*native.Roster, error) { return b.roster, b.err }
func (b *assignmentBackend) SetAssignment(id, list, participant string) (*native.Collaboration, error) {
	b.writes++
	if id != "RESOLVED" || list != "LIST" {
		return nil, errors.New("unresolved identity")
	}
	b.participant = participant
	return &native.Collaboration{AssignmentAvailable: true}, nil
}

func TestAssignResolvesBeforeNativeWrite(t *testing.T) {
	for _, mode := range []string{"assign", "clear", "ambiguous", "missing", "roster-error", "no-list", "conflict", "empty-id"} {
		t.Run(mode, func(t *testing.T) {
			b := &assignmentBackend{roster: &native.Roster{ReminderID: "RESOLVED", ListID: "LIST", People: []native.Participant{{ID: "P", IsMe: true}}}}
			id, query, clear := "prefix", "me", false
			switch mode {
			case "clear":
				query, clear = "", true
			case "ambiguous":
				b.roster.People = append(b.roster.People, native.Participant{ID: "OTHER", IsMe: true})
			case "missing":
				query = "unknown"
			case "roster-error":
				b.err = errors.New("denied")
			case "no-list":
				b.roster.ListID = ""
			case "conflict":
				clear = true
			case "empty-id":
				id = ""
			}
			_, err := (&Client{backend: b}).Assign(id, query, clear)
			valid := mode == "assign" || mode == "clear"
			if (err == nil) != valid {
				t.Fatalf("unexpected result: %v", err)
			}
			if valid {
				want := "P"
				if clear {
					want = ""
				}
				if b.writes != 1 || b.participant != want {
					t.Fatalf("wrong write: %+v", b)
				}
			} else if b.writes != 0 {
				t.Fatal("invalid selector reached native write")
			}
		})
	}
}
