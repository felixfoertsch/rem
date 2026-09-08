package reminderkit

import (
	"testing"

	"github.com/felixfoertsch/rem/internal/reminder"
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

func TestEmptyMailtoCannotMatchMissingAddress(t *testing.T) {
	if _, err := ResolveParticipant([]reminder.Participant{{ID: "P", Name: "Pat"}}, "mailto:"); err == nil {
		t.Fatal("empty mailto matched a missing email")
	}
}
