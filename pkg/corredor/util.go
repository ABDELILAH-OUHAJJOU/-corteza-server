package corredor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"

	"github.com/cortezaproject/corteza-server/pkg/eventbus"
	"github.com/cortezaproject/corteza-server/pkg/slice"
)

type (
	automationListSetPayload struct {
		Filter Filter    `json:"filter"`
		Set    []*Script `json:"set"`
	}
)

// removes onManual event type from trigger
// returns true if event type was removed or
// false if there was no onManual event
func popOnManualEventType(trigger *Trigger) (found bool) {
	for i := len(trigger.EventTypes) - 1; i >= 0; i-- {
		if trigger.EventTypes[i] == onManualEventType {
			found = true

			// remove from the list
			trigger.EventTypes = append(trigger.EventTypes[:i], trigger.EventTypes[i+1:]...)
		}
	}

	return
}

// pluckManualTriggers removes all manual triggers from the list of script's triggers
//
// and returns a hash map with resources from these manual triggers
func pluckManualTriggers(script *ServerScript) map[string]bool {
	var (
		hash = make(map[string]bool)
	)

	for i := range script.Triggers {
		// We're modifying trigger in the loop,
		// so let's make a copy we can play with
		trigger := script.Triggers[i]

		if popOnManualEventType(trigger) {
			for _, res := range trigger.ResourceTypes {
				hash[res] = true
			}
		}
	}

	return hash
}

// converts trigger's constraint to eventbus' constraint options
func makeTriggerOpts(t *Trigger) (oo []eventbus.HandlerRegOp, err error) {
	if len(t.EventTypes) == 0 {
		return nil, fmt.Errorf("can not generate trigger without at least one events")
	}

	if len(t.ResourceTypes) == 0 {
		return nil, fmt.Errorf("can not generate trigger without at least one resource")
	}

	oo = append(oo, eventbus.On(t.EventTypes...))
	oo = append(oo, eventbus.For(t.ResourceTypes...))

	for _, raw := range t.Constraints {
		if c, err := eventbus.ConstraintMaker(raw.Name, raw.Op, raw.Value...); err != nil {
			return nil, errors.Wrap(err, "can not generate trigger")
		} else {
			oo = append(oo, eventbus.Constraint(c))
		}
	}

	return
}

// encode adds entry (with json encoded value) to hash map
// used to prepare data for transmission
func encodeArguments(args map[string]string, key string, val interface{}) (err error) {
	var tmp []byte

	if tmp, err = json.Marshal(val); err != nil {
		return
	}

	args[key] = string(tmp)
	return
}

// Creates a filter fn for script filtering
func makeScriptFilter(f Filter) func(s *Script) (b bool, err error) {
	return func(s *Script) (b bool, err error) {
		b = true
		if len(f.ResourceTypes) > 0 {
			// Filtering by resource type,
			// at least one of the script's triggers should match
			b = false
			for _, t := range s.Triggers {
				if len(slice.IntersectStrings(f.ResourceTypes, t.ResourceTypes)) > 0 {
					b = true
				}
			}

			if !b {
				// No match by resource type, break
				return
			}
		}

		if len(f.EventTypes) > 0 {
			// Filtering by event type,
			// at least one of the script's triggers should match
			b = false
			for _, t := range s.Triggers {
				if len(slice.IntersectStrings(f.EventTypes, t.EventTypes)) > 0 {
					b = true
				}
			}

			if !b {
				// No match by event type, break
				return
			}
		}

		// Not explicitly filtered
		return
	}
}

// GenericListHandler returns filtered list of scripts
func GenericListHandler(ctx context.Context, svc *service, f Filter, resourcePrefix string) (p *automationListSetPayload, err error) {
	f.PrefixResources(resourcePrefix)
	p = &automationListSetPayload{}
	p.Set, p.Filter, err = svc.Find(ctx, f)
	return p, err
}
