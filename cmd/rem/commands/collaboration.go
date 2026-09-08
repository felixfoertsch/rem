package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BRO3886/rem/internal/reminder"
	"github.com/BRO3886/rem/internal/reminderkit"
	"github.com/spf13/cobra"
)

type collaborationBackend interface {
	Participants(string) ([]reminder.Participant, error)
	Assign(string, string, bool) (*reminder.Collaboration, error)
	Sections(string) ([]reminder.Section, error)
	ChangeSection(string, string, string, string) error
	Diagnostics() (map[string]any, error)
}

func writeCollaborationJSON(cmd *cobra.Command, value any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func newAssignCommand(client collaborationBackend) *cobra.Command {
	var none, experimental bool
	cmd := &cobra.Command{
		Use:     "assign <id> [participant]",
		Short:   "Assign a shared reminder to an existing list participant",
		Long:    "Assign by exact participant ID, email, name, or 'me'. Duplicate names are rejected.\nUses private ReminderKit APIs: validate on a disposable shared reminder first.\nThis never invites people, changes sharing permissions, or modifies other reminder fields.",
		Example: "  rem participants --list Family\n  rem assign abc12345 person@example.com --experimental\n  rem assign abc12345 --none --experimental",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("reminder ID is required")
			}
			if (none && len(args) != 1) || (!none && (len(args) != 2 || strings.TrimSpace(args[1]) == "")) {
				return fmt.Errorf("provide a participant or --none, not both")
			}
			if !experimental {
				return fmt.Errorf("native assignment writes require --experimental; test a disposable shared reminder first (see docs/collaboration.md)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			person := ""
			if len(args) == 2 {
				person = args[1]
			}
			result, err := client.Assign(args[0], person, none)
			if err != nil {
				return err
			}
			if result == nil || !result.AssignmentAvailable {
				return fmt.Errorf("assignment result could not be verified")
			}
			if outputFormat == "json" {
				return writeCollaborationJSON(cmd, result)
			}
			if result.AssignedTo == nil {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Assignment cleared.")
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Assigned to %q.\n", result.AssignedTo.String())
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&none, "none", false, "Remove the current assignment")
	cmd.Flags().BoolVar(&experimental, "experimental", false, "Allow native writes pending shared-account validation")
	return cmd
}

func newParticipantsCommand(client collaborationBackend) *cobra.Command {
	var list string
	cmd := &cobra.Command{
		Use: "participants", Short: "List the existing participants of a shared list", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			people, err := client.Participants(list)
			if err != nil {
				return err
			}
			if outputFormat == "json" {
				return writeCollaborationJSON(cmd, people)
			}
			if len(people) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No shared-list participants.")
				return err
			}
			for _, p := range people {
				// Quote externally supplied names/addresses to escape terminal controls.
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%q\t%q\tme=%t\n", p.ID, p.Name, p.Address, p.IsMe); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&list, "list", "l", "", "Exact list name or ID (required)")
	_ = cmd.MarkFlagRequired("list")
	return cmd
}

func newSectionsCommand(client collaborationBackend) *cobra.Command {
	var list string
	cmd := &cobra.Command{
		Use: "sections", Short: "List native sections in a reminder list", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sections, err := client.Sections(list)
			if err != nil {
				return err
			}
			if outputFormat == "json" {
				return writeCollaborationJSON(cmd, sections)
			}
			for _, s := range sections {
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%q\n", s.ID, s.Name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&list, "list", "l", "", "Exact list name or ID (required)")
	_ = cmd.MarkFlagRequired("list")
	return cmd
}

func newSectionCommand(client collaborationBackend) *cobra.Command {
	var list string
	var experimental bool
	parent := &cobra.Command{
		Use: "section", Short: "Create or rename native list sections",
		Long: "Create and rename sections. Moving reminders, filtering by section, and deleting\nsections are not implemented; those require preserving native membership ordering.",
	}
	parent.PersistentFlags().StringVarP(&list, "list", "l", "", "Exact list name or ID (required)")
	parent.PersistentFlags().BoolVar(&experimental, "experimental", false, "Allow native writes pending live-account validation")
	_ = parent.MarkPersistentFlagRequired("list")
	for _, operation := range []string{"create", "rename"} {
		op := operation
		n := 1
		use := op + " <name>"
		if op == "rename" {
			n = 2
			use = op + " <name-or-id> <new-name>"
		}
		parent.AddCommand(&cobra.Command{
			Use: use, Short: op + " a section",
			Args: func(cmd *cobra.Command, args []string) error {
				if err := cobra.ExactArgs(n)(cmd, args); err != nil {
					return err
				}
				if !experimental {
					return fmt.Errorf("native section writes require --experimental; validate on a disposable list first")
				}
				return nil
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				newName := ""
				if op == "rename" {
					newName = args[1]
				}
				if err := client.ChangeSection("section-"+op, list, args[0], newName); err != nil {
					return err
				}
				if outputFormat == "json" {
					return writeCollaborationJSON(cmd, map[string]any{"ok": true, "operation": op})
				}
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Section change saved and verified.")
				return err
			},
		})
	}
	return parent
}

func newDoctorCommand(client collaborationBackend) *cobra.Command {
	return &cobra.Command{
		Use: "doctor", Short: "Show permission status and native capabilities without requesting access", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := client.Diagnostics()
			if err != nil {
				return err
			}
			if result == nil {
				return fmt.Errorf("native diagnostics returned no result")
			}
			result["note"] = "Capability presence is not a live account test. A host application may need its own Reminders permission/entitlement. See docs/troubleshooting.md."
			return writeCollaborationJSON(cmd, result)
		},
	}
}

func init() {
	client := reminderkit.New() // No native store or access request until a data command runs.
	rootCmd.AddCommand(newAssignCommand(client), newParticipantsCommand(client), newSectionsCommand(client), newSectionCommand(client), newDoctorCommand(client))
}
