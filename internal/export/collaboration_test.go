package export

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/felixfoertsch/rem/internal/reminder"
)

func TestAssignmentJSONStatesAndImportIsolation(t *testing.T) {
	for _, state := range []string{"assigned", "unassigned", "unavailable"} {
		t.Run(state, func(t *testing.T) {
			r := &reminder.Reminder{Name: "Task", ListName: "Family", Collaboration: &reminder.Collaboration{AssignmentAvailable: state != "unavailable"}}
			if state == "assigned" {
				r.Collaboration.AssignedTo = &reminder.Participant{ID: "LIST-SCOPED", Name: "Pat"}
			}
			var buf bytes.Buffer
			if err := ExportJSON(&buf, []*reminder.Reminder{r}); err != nil {
				t.Fatal(err)
			}
			var objects []map[string]any
			if err := json.Unmarshal(buf.Bytes(), &objects); err != nil {
				t.Fatal(err)
			}
			if objects[0]["assignment_available"] != (state != "unavailable") {
				t.Fatal("availability lost")
			}
			if _, ok := objects[0]["assigned_to"]; !ok {
				t.Fatal("missing null/assigned property")
			}
			if state != "assigned" && objects[0]["assigned_to"] != nil {
				t.Fatal("expected null assignment")
			}
			imported, err := ImportJSON(&buf)
			if err != nil {
				t.Fatal(err)
			}
			if imported[0].Collaboration != nil {
				t.Fatal("must not replay list-scoped identity on import")
			}
		})
	}
}
