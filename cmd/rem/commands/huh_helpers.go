package commands

import (
	"fmt"
	"os"

	"github.com/felixfoertsch/rem/internal/reminder"
	"github.com/charmbracelet/huh"
)

// huhTheme returns the shared huh theme used across all interactive commands.
func huhTheme() *huh.Theme {
	return huh.ThemeCatppuccin()
}

// isTTY returns true if stdin is a terminal.
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// requireInteractive returns an error if stdin is not a TTY.
func requireInteractive() error {
	if !isTTY() {
		return fmt.Errorf("interactive mode requires a terminal (TTY)")
	}
	return nil
}

// reminderLabel formats a reminder for display in huh selects/multi-selects.
// Format: "Title [List] (due: Jan 02, 15:04) ⚑"
func reminderLabel(r *reminder.Reminder) string {
	title := r.Name
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	label := fmt.Sprintf("%s [%s]", title, r.ListName)

	if r.DueDate != nil {
		label += fmt.Sprintf(" (due: %s)", r.DueDate.Local().Format("Jan 02, 15:04"))
	}

	if r.Flagged {
		label += " ⚑"
	}

	return label
}

// reminderMultiSelect shows a huh multi-select for choosing reminders.
// Returns selected reminder IDs, or nil if user cancelled.
func reminderMultiSelect(title string, reminders []*reminder.Reminder) ([]string, error) {
	if len(reminders) == 0 {
		return nil, fmt.Errorf("no reminders found")
	}

	options := make([]huh.Option[string], len(reminders))
	for i, r := range reminders {
		options[i] = huh.NewOption(reminderLabel(r), r.ID)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(options...).
				Value(&selected).
				Validate(func(s []string) error {
					if len(s) == 0 {
						return fmt.Errorf("select at least one reminder")
					}
					return nil
				}),
		),
	).WithTheme(huhTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			fmt.Println("Cancelled.")
			return nil, nil
		}
		return nil, err
	}

	return selected, nil
}

// reminderSelect shows a huh select for choosing a single reminder.
// Returns the selected reminder ID, or empty string if user cancelled.
func reminderSelect(title string, reminders []*reminder.Reminder) (string, error) {
	if len(reminders) == 0 {
		return "", fmt.Errorf("no reminders found")
	}

	options := make([]huh.Option[string], len(reminders))
	for i, r := range reminders {
		options[i] = huh.NewOption(reminderLabel(r), r.ID)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(options...).
				Value(&selected),
		),
	).WithTheme(huhTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			fmt.Println("Cancelled.")
			return "", nil
		}
		return "", err
	}

	return selected, nil
}

// listSelect fetches all lists and shows a huh select for choosing one.
// Returns the selected list name, or empty string if user cancelled.
func listSelect(title string) (string, error) {
	lists, err := listSvc.GetLists()
	if err != nil {
		return "", err
	}
	if len(lists) == 0 {
		return "", fmt.Errorf("no lists found")
	}

	options := make([]huh.Option[string], len(lists))
	for i, l := range lists {
		options[i] = huh.NewOption(l.Name, l.Name)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(options...).
				Value(&selected),
		),
	).WithTheme(huhTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			fmt.Println("Cancelled.")
			return "", nil
		}
		return "", err
	}

	return selected, nil
}

// huhConfirm shows a huh confirmation prompt.
// Returns (confirmed, error). Returns (false, nil) if user cancelled.
func huhConfirm(title string) (bool, error) {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).WithTheme(huhTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			fmt.Println("Cancelled.")
			return false, nil
		}
		return false, err
	}

	return confirmed, nil
}
