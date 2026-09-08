package reminderkit

// Participant is an existing member of a shared list, not a contact to invite.
// ID is scoped to the list. Never reuse it when moving or importing reminders.
type Participant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address,omitempty"`
	AccessLevel int    `json:"access_level"`
	IsMe        bool   `json:"is_me"`
}

func (p Participant) String() string {
	if p.Name != "" && p.Address != "" {
		return p.Name + " <" + p.Address + ">"
	}
	if p.Name != "" {
		return p.Name
	}
	if p.Address != "" {
		return p.Address
	}
	return p.ID
}

type Section struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// A null AssignedTo means unassigned only when AssignmentAvailable is true.
// An unavailable private API must not be confused with an empty assignment.
type Collaboration struct {
	AssignedTo          *Participant `json:"assigned_to"`
	AssignmentAvailable bool         `json:"assignment_available"`
	AssignmentError     string       `json:"assignment_error,omitempty"`
}
