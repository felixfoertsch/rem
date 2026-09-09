//go:build darwin

package service

import (
	"fmt"

	"github.com/felixfoertsch/rem/go-eventkit/reminders"
	"github.com/felixfoertsch/rem/internal/reminder"
)

// ListService provides operations for reminder lists.
// Uses go-eventkit for all operations (reads and writes).
type ListService struct {
	client *reminders.Client
}

// NewListService creates a new ListService.
func NewListService(client *reminders.Client) *ListService {
	return &ListService{client: client}
}

// GetLists returns all reminder lists via go-eventkit.
func (s *ListService) GetLists() ([]*reminder.List, error) {
	ekLists, err := s.client.Lists()
	if err != nil {
		return nil, fmt.Errorf("failed to get lists: %w", err)
	}

	lists := make([]*reminder.List, 0, len(ekLists))
	for _, l := range ekLists {
		lists = append(lists, fromEventKitList(&l))
	}

	return lists, nil
}

// findListByName looks up a list by name and returns the go-eventkit List.
func (s *ListService) findListByName(name string) (*reminders.List, error) {
	ekLists, err := s.client.Lists()
	if err != nil {
		return nil, fmt.Errorf("failed to get lists: %w", err)
	}

	for _, l := range ekLists {
		if l.Title == name {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("list not found: %s", name)
}

// defaultSource discovers the default source name from existing lists.
// Falls back to "iCloud" if no lists exist.
func (s *ListService) defaultSource() (string, error) {
	ekLists, err := s.client.Lists()
	if err != nil {
		return "", fmt.Errorf("failed to get lists: %w", err)
	}

	for _, l := range ekLists {
		if l.Source != "" {
			return l.Source, nil
		}
	}

	return "iCloud", nil
}

// CreateList creates a new reminder list via go-eventkit.
// The list is created in the default source (discovered from existing lists).
func (s *ListService) CreateList(name string) (*reminder.List, error) {
	if name == "" {
		return nil, fmt.Errorf("list name is required")
	}

	source, err := s.defaultSource()
	if err != nil {
		return nil, err
	}

	created, err := s.client.CreateList(reminders.CreateListInput{
		Title:  name,
		Source: source,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	return fromEventKitList(created), nil
}

// RenameList renames an existing list via go-eventkit.
func (s *ListService) RenameList(oldName, newName string) error {
	ekList, err := s.findListByName(oldName)
	if err != nil {
		return err
	}

	if ekList.ReadOnly {
		return fmt.Errorf("cannot rename list '%s': list is immutable", oldName)
	}

	_, err = s.client.UpdateList(ekList.ID, reminders.UpdateListInput{
		Title: &newName,
	})
	if err != nil {
		return fmt.Errorf("failed to rename list: %w", err)
	}

	return nil
}

// DeleteList deletes a list by name via go-eventkit.
func (s *ListService) DeleteList(name string) error {
	ekList, err := s.findListByName(name)
	if err != nil {
		return err
	}

	if ekList.ReadOnly {
		return fmt.Errorf("cannot delete list '%s': list is immutable", name)
	}

	if err := s.client.DeleteList(ekList.ID); err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
	}

	return nil
}

// fromEventKitList converts a go-eventkit List to an internal List.
func fromEventKitList(l *reminders.List) *reminder.List {
	return &reminder.List{
		ID:          l.ID,
		Name:        l.Title,
		Color:       l.Color,
		Count:       l.Count,
		IsShared:    l.IsShared,
		SharedToMe:  l.SharedToMe,
		IsOwnedByMe: l.IsOwnedByMe,
	}
}
