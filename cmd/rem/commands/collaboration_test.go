package commands

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/BRO3886/rem/internal/reminder"
	"github.com/spf13/cobra"
)

type fakeCollaborationBackend struct {
	calls int
	err   error
	clear bool
}

func (f *fakeCollaborationBackend) Participants(string) ([]reminder.Participant, error) {
	f.calls++
	return []reminder.Participant{{ID: "P", Name: "Pat\x1b[31m", Address: "pat@example.com"}}, f.err
}
func (f *fakeCollaborationBackend) Assign(id, person string, clear bool) (*reminder.Collaboration, error) {
	f.calls++
	f.clear = clear
	if f.err != nil {
		return nil, f.err
	}
	r := &reminder.Collaboration{AssignmentAvailable: true}
	if !clear {
		r.AssignedTo = &reminder.Participant{ID: "P", Name: "Pat"}
	}
	return r, nil
}
func (f *fakeCollaborationBackend) Sections(string) ([]reminder.Section, error) {
	f.calls++
	return []reminder.Section{{ID: "S", Name: "Section"}}, f.err
}
func (f *fakeCollaborationBackend) ChangeSection(string, string, string, string) error {
	f.calls++
	return f.err
}
func (f *fakeCollaborationBackend) Diagnostics() (map[string]any, error) {
	f.calls++
	return map[string]any{"permission_requested": false}, f.err
}
func runCollaborationCommand(cmd *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
func TestAssignRequiresExplicitExperimentalOptIn(t *testing.T) {
	for _, args := range [][]string{{"R", "Pat"}, {"R", "--none"}, {"R", "Pat", "--none", "--experimental"}, {"R", "--experimental"}} {
		backend := &fakeCollaborationBackend{}
		cmd := newAssignCommand(backend)
		cmd.PreRunE = func(*cobra.Command, []string) error {
			t.Fatal("invalid args reached pre-run/permission phase")
			return nil
		}
		if _, err := runCollaborationCommand(cmd, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if backend.calls != 0 {
			t.Fatal("invalid arguments called the native backend")
		}
	}
}
func TestAssignmentCommandOutputAndFailure(t *testing.T) {
	old := outputFormat
	t.Cleanup(func() { outputFormat = old })
	for _, format := range []string{"json", "plain", "table"} {
		outputFormat = format
		for _, clear := range []bool{false, true} {
			backend := &fakeCollaborationBackend{}
			args := []string{"R", "Pat", "--experimental"}
			if clear {
				args = []string{"R", "--none", "--experimental"}
			}
			out, err := runCollaborationCommand(newAssignCommand(backend), args...)
			if err != nil || backend.calls != 1 || backend.clear != clear {
				t.Fatalf("%s: %q %v", format, out, err)
			}
			want := "Assigned to"
			if clear {
				want = "Assignment cleared"
			}
			if format == "json" {
				want = `"assignment_available": true`
			}
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q in %q", want, out)
			}
		}
		backend := &fakeCollaborationBackend{err: errors.New("save failed")}
		out, err := runCollaborationCommand(newAssignCommand(backend), "R", "Pat", "--experimental")
		if err == nil || out != "" {
			t.Fatalf("failure emitted success output %q, %v", out, err)
		}
	}
}
func TestCollaborationHelpNeverCallsBackend(t *testing.T) {
	backend := &fakeCollaborationBackend{err: errors.New("must not call")}
	for _, cmd := range []*cobra.Command{newAssignCommand(backend), newParticipantsCommand(backend), newSectionsCommand(backend), newSectionCommand(backend), newDoctorCommand(backend)} {
		if _, err := runCollaborationCommand(cmd, "--help"); err != nil {
			t.Fatal(err)
		}
	}
	if backend.calls != 0 {
		t.Fatal("help invoked native API")
	}
}
func TestSectionWritesRequireOptInAndPropagateErrors(t *testing.T) {
	for _, args := range [][]string{{"create", "S", "--list", "L"}, {"rename", "S", "T", "--list", "L"}} {
		backend := &fakeCollaborationBackend{}
		if _, err := runCollaborationCommand(newSectionCommand(backend), args...); err == nil || backend.calls != 0 {
			t.Fatal("section mutation without opt-in")
		}
		backend.err = errors.New("section save failed")
		out, err := runCollaborationCommand(newSectionCommand(backend), append(args, "--experimental")...)
		if err == nil || out != "" || backend.calls != 1 {
			t.Fatalf("%q %v", out, err)
		}
	}
}
func TestParticipantsEscapeTerminalControls(t *testing.T) {
	old := outputFormat
	outputFormat = "plain"
	t.Cleanup(func() { outputFormat = old })
	out, err := runCollaborationCommand(newParticipantsCommand(&fakeCollaborationBackend{}), "--list", "Family")
	if err != nil || strings.Contains(out, "\x1b") || !strings.Contains(out, "Pat\\x1b") {
		t.Fatalf("unsafe output %q: %v", out, err)
	}
}
