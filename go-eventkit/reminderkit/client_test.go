package reminderkit

import (
	"encoding/json"
	"testing"
)

func fill(out any, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func TestInvalidRequestsNeverCrossBridge(t *testing.T) {
	c := &Client{call: func(any, any) error { t.Fatal("invalid request crossed native boundary"); return nil }}
	if data, err := c.Metadata(nil); err != nil || len(data) != 0 {
		t.Fatalf("empty metadata: %v %v", data, err)
	}
	if _, err := c.Roster(" "); err == nil {
		t.Fatal("blank reminder accepted")
	}
	for _, args := range [][4]string{{"section-delete", "L", "S", ""}, {"section-create", "", "S", ""}, {"section-create", "L", "", ""}, {"section-rename", "L", "S", ""}} {
		if err := c.ChangeSection(args[0], args[1], args[2], args[3]); err == nil {
			t.Fatalf("invalid section request accepted: %v", args)
		}
	}
}

func TestSectionResultMustBeVerified(t *testing.T) {
	for _, op := range []string{"section-create", "section-rename"} {
		for _, valid := range []bool{false, true} {
			c := &Client{call: func(request, out any) error {
				r := request.(map[string]any)
				name := r["name"].(string)
				if op == "section-rename" {
					name = r["new_name"].(string)
				}
				if valid {
					return fill(out, Section{ID: "S", Name: name})
				}
				return fill(out, nil)
			}}
			if err := c.ChangeSection(op, "L", "Section", "New name"); (err == nil) != valid {
				t.Fatalf("%s valid=%t: %v", op, valid, err)
			}
		}
	}
}
