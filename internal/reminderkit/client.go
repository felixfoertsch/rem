// Package reminderkit resolves CLI participant selectors before calling go-eventkit.
// Native transactions and verification belong to the library.
package reminderkit

import (
	"fmt"
	"strings"

	native "github.com/felixfoertsch/rem/go-eventkit/reminderkit"
	"github.com/felixfoertsch/rem/internal/reminder"
)

type backend interface {
	Roster(string) (*native.Roster, error)
	SetAssignment(string, string, string) (*native.Collaboration, error)
	Participants(string) ([]native.Participant, error)
	Metadata([]string) (map[string]*native.Collaboration, error)
	Sections(string) ([]native.Section, error)
	ChangeSection(string, string, string, string) error
	Diagnostics() (map[string]any, error)
}

type Client struct{ backend }

func New() *Client { return &Client{backend: native.New()} }

// ResolveParticipant never performs prefix/fuzzy matching on people's names.
// Exact IDs win; names and email addresses are case-insensitive. "me" requires
// native current-user identity, never the shell username or a contact guess.
func ResolveParticipant(people []reminder.Participant, query string) (*reminder.Participant, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("participant is required")
	}
	for _, p := range people {
		if strings.EqualFold(p.ID, query) && p.ID != "" {
			v := p
			return &v, nil
		}
	}
	var found *reminder.Participant
	for _, p := range people {
		match := strings.EqualFold(query, "me") && p.IsMe
		if !strings.EqualFold(query, "me") {
			match = strings.EqualFold(p.Name, query) || (trimMailto(p.Address) != "" && strings.EqualFold(trimMailto(p.Address), trimMailto(query)))
		}
		if !match {
			continue
		}
		if found != nil && found.ID != p.ID {
			return nil, fmt.Errorf("participant %q is ambiguous; use an exact email address or participant ID from rem participants", query)
		}
		v := p
		found = &v
	}
	if found == nil {
		return nil, fmt.Errorf("participant %q not found in this shared list; run rem participants --list <list>", query)
	}
	if found.ID == "" {
		return nil, fmt.Errorf("participant %q has no usable native ID", query)
	}
	return found, nil
}

func trimMailto(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 7 && strings.EqualFold(s[:7], "mailto:") {
		return s[7:]
	}
	return s
}

// Assign first resolves the roster, then revalidates it when preparing the native transaction.
// An empty query is allowed only when explicitly clearing the assignment.
func (c *Client) Assign(id, query string, clear bool) (*reminder.Collaboration, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("reminder ID is required")
	}
	if clear == (strings.TrimSpace(query) != "") {
		return nil, fmt.Errorf("provide a participant or --none, not both")
	}
	roster, err := c.Roster(id)
	if err != nil {
		return nil, err
	}
	if roster == nil || roster.ListID == "" || roster.ReminderID == "" {
		return nil, fmt.Errorf("native roster has no resolved reminder/list ID; refusing to write")
	}
	participantID := ""
	if !clear {
		p, err := ResolveParticipant(roster.People, query)
		if err != nil {
			return nil, err
		}
		participantID = p.ID
	}
	return c.SetAssignment(roster.ReminderID, roster.ListID, participantID)
}
