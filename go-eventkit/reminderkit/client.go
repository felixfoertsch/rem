// Package reminderkit provides guarded native collaboration APIs for Reminders.
// Construction never requests access. Writes use REMSaveRequest and local readback;
// successful local verification does not establish remote synchronization.
package reminderkit

import (
	"fmt"
	"strings"
)

type Client struct{ call func(any, any) error }

func New() *Client { return &Client{call: nativeCall} }

// Roster pins the resolved reminder and list for a subsequent assignment.
type Roster struct {
	People     []Participant `json:"people"`
	ListID     string        `json:"list_id"`
	ReminderID string        `json:"reminder_id"`
}

func (c *Client) Participants(list string) ([]Participant, error) {
	out := []Participant{}
	err := c.call(map[string]any{"op": "participants", "list": list}, &out)
	return out, err
}

func (c *Client) Roster(id string) (*Roster, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("reminder ID is required")
	}
	var out Roster
	if err := c.call(map[string]any{"op": "roster", "id": id}, &out); err != nil {
		return nil, err
	}
	if out.ListID == "" || out.ReminderID == "" {
		return nil, fmt.Errorf("native roster has no resolved reminder/list ID; refusing to write")
	}
	return &out, nil
}

// SetAssignment accepts resolved native IDs, not names, emails or CLI selectors.
// Empty participantID explicitly clears assignment. Native code revalidates the
// reminder's list and participant membership immediately before saving.
func (c *Client) SetAssignment(reminderID, listID, participantID string) (*Collaboration, error) {
	if strings.TrimSpace(reminderID) == "" || strings.TrimSpace(listID) == "" {
		return nil, fmt.Errorf("resolved reminder and list IDs are required")
	}
	if participantID != "" && strings.TrimSpace(participantID) == "" {
		return nil, fmt.Errorf("participant ID must not be blank")
	}
	clear := participantID == ""
	var out Collaboration
	err := c.call(map[string]any{"op": "assign", "id": reminderID, "list_id": listID, "participant_id": participantID, "clear": clear}, &out)
	if err != nil {
		return nil, err
	}
	if !out.AssignmentAvailable || (clear && out.AssignedTo != nil) || (!clear && (out.AssignedTo == nil || !strings.EqualFold(out.AssignedTo.ID, participantID))) {
		return nil, fmt.Errorf("assignment operation returned an unverified result; inspect Reminders before retrying")
	}
	return &out, nil
}

func (c *Client) Metadata(ids []string) (map[string]*Collaboration, error) {
	out := make(map[string]*Collaboration)
	if len(ids) == 0 {
		return out, nil
	}
	err := c.call(map[string]any{"op": "metadata", "ids": ids}, &out)
	return out, err
}

func (c *Client) Sections(list string) ([]Section, error) {
	out := []Section{}
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
	var out Section
	if err := c.call(map[string]any{"op": op, "list": list, "name": name, "new_name": newName}, &out); err != nil {
		return err
	}
	want := newName
	if op == "section-create" {
		want = name
	}
	if out.ID == "" || out.Name != want {
		return fmt.Errorf("section result could not be verified; inspect Reminders before retrying")
	}
	return nil
}

func (c *Client) Diagnostics() (map[string]any, error) {
	out := make(map[string]any)
	err := c.call(map[string]any{"op": "diagnostics"}, &out)
	if err == nil && out == nil {
		err = fmt.Errorf("native diagnostics returned no result")
	}
	return out, err
}
