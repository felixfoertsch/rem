package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/felixfoertsch/rem/internal/reminder"
)

func TestAssignmentDisplay(t *testing.T) {
	r := &reminder.Reminder{ID: "R", Name: "Task", ListName: "Family"}
	if assignmentLabel(r) != "unavailable" {
		t.Fatal("missing metadata misreported")
	}
	r.Collaboration = &reminder.Collaboration{AssignmentAvailable: true}
	if assignmentLabel(r) != "unassigned" {
		t.Fatal("unassigned display")
	}
	r.Collaboration.AssignedTo = &reminder.Participant{ID: "P", Name: "Pat\x1b[2J"}
	if strings.Contains(assignmentLabel(r), "\x1b") {
		t.Fatal("unescaped terminal control")
	}
	for _, format := range []OutputFormat{FormatTable, FormatPlain, FormatJSON} {
		var b bytes.Buffer
		PrintReminderDetail(&b, r, format)
		if !strings.Contains(b.String(), "Pat") {
			t.Fatalf("assignment missing in %s detail", format)
		}
		b.Reset()
		PrintReminders(&b, []*reminder.Reminder{r}, format)
		if !strings.Contains(b.String(), "Pat") {
			t.Fatalf("assignment missing in %s list", format)
		}
	}
}
