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
	calls       int
	err         error
	clear       bool
	id, person  string
	sectionArgs [4]string
}

func (f *fakeCollaborationBackend) Participants(string) ([]reminder.Participant, error) {
	f.calls++
	return []reminder.Participant{{ID: "P", Name: "Pat\x1b[31m", Address: "pat@example.com"}}, f.err
}
func (f *fakeCollaborationBackend) Assign(id, person string, clear bool) (*reminder.Collaboration, error) {
	f.calls++
	f.clear = clear
	f.id, f.person = id, person
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
func (f *fakeCollaborationBackend) ChangeSection(op, list, name, newName string) error {
	f.calls++
	f.sectionArgs = [4]string{op, list, name, newName}
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
func TestAssignRejectsInvalidArgumentsBeforeAccess(t *testing.T) {
	for _, args := range [][]string{
		{}, {"R"}, {"", "Pat"}, {" ", "Pat"}, {"R", ""}, {"R", " "},
		{"R", "Pat", "--none"}, {"R", "Pat", "extra"},
	} {
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
		for _, person := range []string{"P", "Pat", "pat@example.com", "me", ""} {
			clear := person == ""
			backend := &fakeCollaborationBackend{}
			args := []string{"R", person}
			if clear {
				args = []string{"R", "--none"}
			}
			out, err := runCollaborationCommand(newAssignCommand(backend), args...)
			if err != nil || backend.calls != 1 || backend.clear != clear || backend.id != "R" || backend.person != person {
				t.Fatalf("%s, %v: %q %v, backend=%+v", format, args, out, err, backend)
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
		out, err := runCollaborationCommand(newAssignCommand(backend), "R", "Pat")
		if !errors.Is(err, backend.err) || out != "" || backend.calls != 1 {
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
func TestCollaborationCommandsHaveNoExperimentalFlag(t *testing.T) {
	for _, args := range [][]string{{"assign", "--help"}, {"section", "--help"}, {"section", "create", "--help"}, {"section", "rename", "--help"}} {
		backend := &fakeCollaborationBackend{}
		root := &cobra.Command{Use: "rem"}
		root.AddCommand(newAssignCommand(backend), newSectionCommand(backend))
		out, err := runCollaborationCommand(root, args...)
		if err != nil || strings.Contains(out, "--experimental") || backend.calls != 0 {
			t.Fatalf("%v: unexpected help %q, %v", args, out, err)
		}
		cmd, _, err := root.Find(args[:len(args)-1])
		if err != nil || cmd.Flags().Lookup("experimental") != nil || cmd.InheritedFlags().Lookup("experimental") != nil {
			t.Fatalf("%v: experimental flag is still registered, %v", args, err)
		}
	}
}
func TestSectionWritesWorkByDefaultAndPropagateErrors(t *testing.T) {
	old := outputFormat
	t.Cleanup(func() { outputFormat = old })
	for _, format := range []string{"json", "plain", "table"} {
		outputFormat = format
		for _, tt := range []struct {
			args []string
			want [4]string
		}{
			{[]string{"create", "S", "--list", "L"}, [4]string{"section-create", "L", "S", ""}},
			{[]string{"rename", "S", "T", "--list", "L"}, [4]string{"section-rename", "L", "S", "T"}},
		} {
			backend := &fakeCollaborationBackend{}
			out, err := runCollaborationCommand(newSectionCommand(backend), tt.args...)
			if err != nil || out == "" || backend.calls != 1 || backend.sectionArgs != tt.want {
				t.Fatalf("%v: %q %v, backend=%+v", tt.args, out, err, backend)
			}
			backend = &fakeCollaborationBackend{err: errors.New("section save failed")}
			out, err = runCollaborationCommand(newSectionCommand(backend), tt.args...)
			if !errors.Is(err, backend.err) || out != "" || backend.calls != 1 {
				t.Fatalf("%q %v", out, err)
			}
		}
	}
}
func TestSectionWritesStillValidateArguments(t *testing.T) {
	for _, args := range [][]string{
		{"create", "--list", "L"}, {"create", "S", "T", "--list", "L"},
		{"rename", "S", "--list", "L"}, {"rename", "S", "T", "extra", "--list", "L"},
		{"create", "S"}, {"rename", "S", "T"},
	} {
		backend := &fakeCollaborationBackend{}
		if _, err := runCollaborationCommand(newSectionCommand(backend), args...); err == nil || backend.calls != 0 {
			t.Fatalf("invalid section arguments accepted: %v", args)
		}
	}
}
func TestCollaborationWritesStillRequireRemindersAccess(t *testing.T) {
	original, format := initializeServices, outputFormat
	t.Cleanup(func() { initializeServices, outputFormat = original, format })
	outputFormat = "json"
	denied := errors.New("permission denied in test")
	for _, args := range [][]string{
		{"assign", "R", "Pat"}, {"assign", "R", "--none"},
		{"section", "create", "S", "--list", "L"},
		{"section", "rename", "S", "T", "--list", "L"},
	} {
		calls := 0
		initializeServices = func() error { calls++; return denied }
		backend := &fakeCollaborationBackend{}
		root := &cobra.Command{Use: "rem", PersistentPreRunE: rootCmd.PersistentPreRunE}
		root.AddCommand(newAssignCommand(backend), newSectionCommand(backend))
		out, err := runCollaborationCommand(root, args...)
		if !errors.Is(err, denied) || out != "" || calls != 1 || backend.calls != 0 {
			t.Fatalf("%v: wanted access denial before native write, got %q %v, access=%d native=%d", args, out, err, calls, backend.calls)
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
