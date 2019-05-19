package types

import (
	"testing"

	"errors"

	"github.com/crusttech/crust/internal/test"
)

// 	Hello! This file is auto-generated.

func TestUserSetWalk(t *testing.T) {
	value := make(UserSet, 3)

	// check walk with no errors
	{
		err := value.Walk(func(*User) error {
			return nil
		})
		test.NoError(t, err, "Expected no returned error from Walk, got %+v", err)
	}

	// check walk with error
	test.Error(t, value.Walk(func(*User) error { return errors.New("Walk error") }), "Expected error from walk, got nil")
}

func TestUserSetFilter(t *testing.T) {
	value := make(UserSet, 3)

	// filter nothing
	{
		set, err := value.Filter(func(*User) (bool, error) {
			return true, nil
		})
		test.NoError(t, err, "Didn't expect error when filtering set: %+v", err)
		test.Assert(t, len(set) == len(value), "Expected equal length filter: %d != %d", len(value), len(set))
	}

	// filter one item
	{
		found := false
		set, err := value.Filter(func(*User) (bool, error) {
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
		_, err := value.Filter(func(*User) (bool, error) {
			return false, errors.New("Filter error")
		})
		test.Error(t, err, "Expected error, got %#v", err)
	}
}

func TestUserSetIDs(t *testing.T) {
	value := make(UserSet, 3)
	// construct objects
	value[0] = new(User)
	value[1] = new(User)
	value[2] = new(User)
	// set ids
	value[0].ID = 1
	value[1].ID = 2
	value[2].ID = 3

	// Find existing
	{
		val := value.FindByID(2)
		test.Assert(t, val.ID == 2, "Expected ID 2, got %d", val.ID)
	}

	// Find non-existing
	{
		val := value.FindByID(4)
		test.Assert(t, val == nil, "Expected no value, got %#v", val)
	}

	// List IDs from set
	{
		val := value.IDs()
		test.Assert(t, len(val) == len(value), "Expected ID count mismatch, %d != %d", len(val), len(value))
	}
}
