package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/felixfoertsch/rem/internal/reminder"
	"github.com/felixfoertsch/rem/internal/ui"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's and overdue reminders",
	Long:  `Show all incomplete reminders that are due today or overdue.`,
	Example: `  rem today
  rem today --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		completed := false
		filter := &reminder.ListFilter{
			ListName:  todayList,
			Completed: &completed,
			DueBefore: &endOfToday,
		}

		all, err := reminderSvc.ListReminders(filter)
		if err != nil {
			return err
		}

		// Filter out reminders with no due date (DueBefore may include them
		// depending on EventKit behavior)
		var reminders []*reminder.Reminder
		for _, r := range all {
			if r.DueDate != nil {
				reminders = append(reminders, r)
			}
		}

		format := ui.ParseOutputFormat(outputFormat)
		if len(reminders) == 0 {
			if format != ui.FormatJSON {
				fmt.Println("No reminders due today!")
			} else {
				fmt.Println("[]")
			}
			return nil
		}

		ui.PrintReminders(os.Stdout, reminders, format)
		return nil
	},
}

var todayList string

func init() {
	todayCmd.Flags().StringVarP(&todayList, "list", "l", "", "Filter by list name")
	rootCmd.AddCommand(todayCmd)
}
