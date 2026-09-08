// Package reminderkit isolates native capabilities not yet exposed by
// go-eventkit. It never grants access or modifies the Reminders database
// directly: all writes go through Apple's REMSaveRequest transaction API.
package reminderkit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BRO3886/rem/internal/reminder"
)

type Client struct {
	call func(any, any) error
}

func New() *Client { return &Client{call: nativeCall} }

// Participants returns existing participants, including the list owner.
func (c *Client) Participants(list string) ([]reminder.Participant, error) {
	out := []reminder.Participant{}
	err := c.call(map[string]any{"op": "participants", "list": list}, &out)
	return out, err
}

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
			match = strings.EqualFold(p.Name, query) || strings.EqualFold(trimMailto(p.Address), trimMailto(query))
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
	var roster struct {
		People []reminder.Participant `json:"people"`
		ListID string                 `json:"list_id"`
	}
	if err := c.call(map[string]any{"op": "roster", "id": id}, &roster); err != nil {
		return nil, err
	}
	participantID := ""
	if !clear {
		p, err := ResolveParticipant(roster.People, query)
		if err != nil {
			return nil, err
		}
		participantID = p.ID
	}
	var out reminder.Collaboration
	if roster.ListID == "" {
		return nil, fmt.Errorf("native roster has no list ID; refusing to write")
	}
	err := c.call(map[string]any{"op": "assign", "id": id, "list_id": roster.ListID, "participant_id": participantID, "clear": clear}, &out)
	if err != nil {
		return nil, err
	}
	if !out.AssignmentAvailable || (clear && out.AssignedTo != nil) || (!clear && (out.AssignedTo == nil || !strings.EqualFold(out.AssignedTo.ID, participantID))) {
		return nil, fmt.Errorf("assignment operation returned an unverified result; inspect Reminders before retrying")
	}
	return &out, nil
}

func (c *Client) Metadata(ids []string) (map[string]*reminder.Collaboration, error) {
	out := make(map[string]*reminder.Collaboration)
	if len(ids) == 0 {
		return out, nil
	}
	err := c.call(map[string]any{"op": "metadata", "ids": ids}, &out)
	return out, err
}

func (c *Client) Sections(list string) ([]reminder.Section, error) {
	out := []reminder.Section{}
	err := c.call(map[string]any{"op": "sections", "list": list}, &out)
	return out, err
}

func (c *Client) ChangeSection(op, list, name, newName string) error {
	switch op {
	case "section-create", "section-rename":
	default:
		return fmt.Errorf("invalid section operation %q", op)
	}
	if strings.TrimSpace(list) == "" || strings.TrimSpace(name) == "" {
		return fmt.Errorf("list and section are required")
	}
	if op == "section-rename" && strings.TrimSpace(newName) == "" {
		return fmt.Errorf("new section name is required")
	}
	var out json.RawMessage
	return c.call(map[string]any{"op": op, "list": list, "name": name, "new_name": newName}, &out)
}

// Diagnostics does not instantiate an EventKit/REMStore or request permission.
func (c *Client) Diagnostics() (map[string]any, error) {
	out := make(map[string]any)
	err := c.call(map[string]any{"op": "diagnostics"}, &out)
	return out, err
}
