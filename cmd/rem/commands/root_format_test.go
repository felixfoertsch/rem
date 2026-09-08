package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestOutputFormatCompatibility(t *testing.T) {
	old := outputFormat
	t.Cleanup(func() { outputFormat = old })
	for input, want := range map[string]string{"JSON": "json", "text": "plain", "PLAIN": "plain", "table": "table"} {
		outputFormat = input
		if err := rootCmd.PersistentPreRunE(&cobra.Command{Use: "doctor"}, nil); err != nil {
			t.Fatal(err)
		}
		if outputFormat != want {
			t.Fatalf("%q normalized to %q, want %q", input, outputFormat, want)
		}
	}
	outputFormat = "invalid"
	if err := rootCmd.PersistentPreRunE(&cobra.Command{Use: "doctor"}, nil); err == nil {
		t.Fatal("invalid output format accepted")
	}
}
