package ui

import (
	"strconv"

	"github.com/felixfoertsch/rem/internal/reminder"
)

func assignmentLabel(r *reminder.Reminder) string {
	if r.Collaboration == nil || !r.Collaboration.AssignmentAvailable {
		return "unavailable"
	}
	if r.Collaboration.AssignedTo == nil {
		return "unassigned"
	}
	// Participant data is supplied by other users of a shared list. Escape
	// terminal controls rather than interpreting them as ANSI sequences.
	return strconv.Quote(r.Collaboration.AssignedTo.String())
}
