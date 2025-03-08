package pidp11

import (
	"fmt"
)

// The mechanism for detecting knob rotations used in the original C version
// can be simplified to only check for a sequence of 2 non-consecutive events,
// instead of a sequence of 4 consecutive events.

type knobEvent struct {
	isCw, state bool
}

// First and last events for clockwise/anticlockwise rotation
var cwStart = knobEvent{true, true}
var cwEnd = knobEvent{false, false}
var acwStart = knobEvent{false, true}
var acwEnd = knobEvent{true, false}
var end knobEvent

// For non-knobs, simply returns the event.
// For knobs returns either:
//   - an event with ID SS_NIL, meaning the event should be ignored
//   - or a synthetic event with ID SW_KNOBA or SW_KNOBD, and a state
//     indicating the direction of the rotation.
func eventForKnob(nid nativeSwitchID, state bool) Event {
	var knobID SwitchID
	var isCw bool
	switch nid {
	case swKNOBA_CW:
		knobID = SS_KNOBA
		isCw = true
	case swKNOBA_ACW:
		knobID = SS_KNOBA
		isCw = false
	case swKNOBD_CW:
		knobID = SS_KNOBD
		isCw = true
	case swKNOBD_ACW:
		knobID = SS_KNOBD
		isCw = false
	default:
		panic(fmt.Errorf("not a knob ID: %d", nid))
	}
	kev := knobEvent{isCw, state}
	switch kev {
	case cwStart:
		end = cwEnd
		return Event{}
	case acwStart:
		end = acwEnd
		return Event{}
	case end:
		return Event{ID: knobID, On: end == cwEnd}
	default:
		return Event{}
	}
}
