package types

import (
	"testing"

	"errors"

	"github.com/crusttech/crust/internal/test"
)

// 	Hello! This file is auto-generated.

func TestUnreadSetWalk(t *testing.T) {
	value := make(UnreadSet, 3)

	// check walk with no errors
	{
		err := value.Walk(func(*Unread) error {
			return nil
		})
		test.NoError(t, err, "Expected no returned error from Walk, got %+v", err)
	}

	// check walk with error
	test.Error(t, value.Walk(func(*Unread) error { return errors.New("Walk error") }), "Expected error from walk, got nil")
}

func TestUnreadSetFilter(t *testing.T) {
	value := make(UnreadSet, 3)

	// filter nothing
	{
		set, err := value.Filter(func(*Unread) (bool, error) {
			return true, nil
		})
		test.NoError(t, err, "Didn't expect error when filtering set: %+v", err)
		test.Assert(t, len(set) == len(value), "Expected equal length filter: %d != %d", len(value), len(set))
	}

	// filter one item
	{
		found := false
		set, err := value.Filter(func(*Unread) (bool, error) {
			if !found {
				found = true
				return found, nil
			}
			return false, nil
		})
		test.NoError(t, err, "Didn't expect error when filtering set: %+v", err)
		test.Assert(t, len(set) == 1, "Expected single item, got %d", len(value))
	}

	// filter error
	{
		_, err := value.Filter(func(*Unread) (bool, error) {
			return false, errors.New("Filter error")
		})
		test.Error(t, err, "Expected error, got %#v", err)
	}
}
