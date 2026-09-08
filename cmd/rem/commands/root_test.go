package commands

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandNeedsReminders(t *testing.T) {
	for _, name := range []string{"help", "version", "skills", "completion", "__complete", "__completeNoDesc", "doctor"} {
		t.Run(name, func(t *testing.T) {
			parent := &cobra.Command{Use: name}
			child := &cobra.Command{Use: "child"}
			parent.AddCommand(child)
			if commandNeedsReminders(parent) || commandNeedsReminders(child) {
				t.Fatal("offline commands and their children must not request access")
			}
		})
	}
	for _, name := range []string{"list", "add", "update", "assign", "sections"} {
		if !commandNeedsReminders(&cobra.Command{Use: name}) {
			t.Errorf("%s must initialize access", name)
		}
	}
}

func TestPreRunDefersAccessUntilNeeded(t *testing.T) {
	original, format := initializeServices, outputFormat
	t.Cleanup(func() { initializeServices, outputFormat = original, format })
	outputFormat = "json"
	denied := errors.New("permission denied in test")
	calls := 0
	initializeServices = func() error { calls++; return denied }
	for _, name := range []string{"version", "skills", "completion", "doctor"} {
		if err := rootCmd.PersistentPreRunE(&cobra.Command{Use: name}, nil); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if calls != 0 {
		t.Fatalf("offline commands requested access %d times", calls)
	}
	if err := rootCmd.PersistentPreRunE(&cobra.Command{Use: "list"}, nil); !errors.Is(err, denied) {
		t.Fatalf("wanted the access error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("wanted one access request, got %d", calls)
	}
}
