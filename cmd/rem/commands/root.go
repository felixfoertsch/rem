package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/BRO3886/go-eventkit/reminders"
	"github.com/BRO3886/rem/internal/service"
	"github.com/BRO3886/rem/internal/skills"
	"github.com/BRO3886/rem/internal/update"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
	noColor      bool

	exec        *service.Executor
	reminderSvc *service.ReminderService
	listSvc     *service.ListService
)

// initializeServices is deliberately lazy: help, skills, completions and unit
// tests must never open EventKit or trigger a macOS permission prompt.
// The function variable also provides a seam for permission regression tests.
var initializeServices = func() error {
	if reminderSvc != nil && listSvc != nil {
		return nil
	}
	client, err := reminders.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Reminders access: %w\n\nRun rem from Terminal.app and allow Reminders access when prompted. Check System Settings > Privacy & Security > Reminders for the application launching rem. A grant to one terminal does not necessarily apply to an IDE or agent application. If the host cannot request access, rem cannot grant it on the host's behalf. See docs/troubleshooting.md", err)
	}
	exec = service.NewExecutor()
	reminderSvc = service.NewReminderService(client)
	listSvc = service.NewListService(client, exec)
	return nil
}

// Each invocation owns its channel, including when Execute is called again in
// tests. A late update check cannot block or poison a subsequent invocation.
var updateResultCh chan *update.Result

var rootCmd = &cobra.Command{
	Use:   "rem",
	Short: "A powerful CLI for macOS Reminders",
	Long: `rem is a command-line interface for interacting with the macOS Reminders app.
It provides full CRUD operations for reminders and lists, natural language date parsing,
import/export capabilities, and a clean terminal UI.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		outputFormat = strings.ToLower(strings.TrimSpace(outputFormat))
		if outputFormat == "text" {
			outputFormat = "plain"
		}
		switch outputFormat {
		case "table", "json", "plain":
		default:
			return fmt.Errorf("invalid output format %q: use table, json, or plain", outputFormat)
		}
		if noColor || os.Getenv("NO_COLOR") != "" {
			color.NoColor = true
		}
		if commandNeedsReminders(cmd) {
			if err := initializeServices(); err != nil {
				return err
			}
		}

		ch := make(chan *update.Result, 1)
		updateResultCh = ch
		if shouldCheckForUpdate(cmd) {
			go func() {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					ch <- nil
					return
				}
				ch <- update.Check(homeDir, Version)
			}()
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if commandNeedsReminders(cmd) {
			printUpdateNotice(cmd)
		}
	},
}

// Check ancestors as well: "skills status" and "completion bash" must be as
// permission-free as their parent commands. Cobra's hidden completion commands
// also run without touching the database.
func commandNeedsReminders(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "version", "help", "skills", "completion", "__complete", "__completeNoDesc", "doctor":
			return false
		}
	}
	return true
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json, plain")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func shouldCheckForUpdate(cmd *cobra.Command) bool {
	if !commandNeedsReminders(cmd) || os.Getenv("REM_NO_UPDATE_CHECK") != "" {
		return false
	}
	if Version == "" || Version == "dev" || outputFormat == "json" {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// printUpdateNotice prints update and skills staleness notices to stderr.
func printUpdateNotice(_ *cobra.Command) {
	var result *update.Result
	select {
	case result = <-updateResultCh:
	default:
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	yellow := color.New(color.FgYellow)
	if result != nil && result.HasUpdate {
		fmt.Fprintln(os.Stderr)
		yellow.Fprintf(os.Stderr, "A new version of rem is available: %s → %s\n", Version, result.Latest)
		fmt.Fprintf(os.Stderr, "Update: curl -fsSL https://rem.sidv.dev/install | bash\n")
	}
	printSkillsStalenessNotice(homeDir)
}

func printSkillsStalenessNotice(homeDir string) {
	if Version == "" || Version == "dev" {
		return
	}
	targets := skills.InstalledTargets(skills.DefaultTargets(homeDir))
	for _, t := range targets {
		installed := skills.InstalledVersion(t)
		if installed != "" && installed != Version {
			yellow := color.New(color.FgYellow)
			fmt.Fprintln(os.Stderr)
			yellow.Fprintf(os.Stderr, "Installed skills are outdated (%s). Run: rem skills install\n", installed)
			return
		}
	}
}
