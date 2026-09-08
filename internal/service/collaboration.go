//go:build darwin

package service

import (
	"fmt"
	"os"

	"github.com/felixfoertsch/rem/internal/reminder"
)

type collaborationReader interface {
	Metadata([]string) (map[string]*reminder.Collaboration, error)
}

// Enrichment is best-effort and batched. It must never turn a core read into a
// failure, or silently label unavailable assignment data as "unassigned".
func (s *ReminderService) enrichCollaboration(items []*reminder.Reminder) {
	if s.collaboration == nil || len(items) == 0 {
		return
	}
	ids := make([]string, 0, len(items))
	for _, r := range items {
		ids = append(ids, r.ID)
	}
	metadata, err := s.collaboration.Metadata(ids)
	warning := ""
	for _, r := range items {
		if err != nil {
			r.Collaboration = &reminder.Collaboration{AssignmentError: err.Error()}
		} else {
			r.Collaboration = metadata[r.ID]
			if r.Collaboration == nil {
				r.Collaboration = &reminder.Collaboration{AssignmentError: "Native metadata was not returned"}
			}
		}
		if !r.Collaboration.AssignmentAvailable && warning == "" {
			warning = r.Collaboration.AssignmentError
			if warning == "" {
				warning = "Private assignment API unavailable"
			}
		}
	}
	if warning != "" {
		fmt.Fprintf(os.Stderr, "Warning: some assignment metadata is unavailable: %s\n", warning)
	}
}
