package pidp11

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

type LedID int
type SwitchID int
type nativeSwitchID int

type Event struct {
	ID SwitchID
	On bool
}

var NON_EVENT = Event{ID: SS_NIL}

var events chan Event
var running bool
var ledSpecs [72]ledSpec
var switches [38]bool        // current state per nativeSwitchID
var brightnessAdjust float64 // adjust the max brightness for all leds
var brigthnessScaler Scaler
var frequencyScaler Scaler
var loopμs int // approx. duration of a loop, for converting durations to loops
var logger *slog.Logger

// Current brightness of the led, and envelope
type ledSpec struct {
	sync.Mutex
	bright int // brightness, 0-31
	env    envelope
	name   string // for debug messages
}

func Start(logger0 *slog.Logger) error {
	logger = logger0
	events = make(chan Event, 100)

	// Sensible defaults
	if brightnessAdjust == 0 {
		brightnessAdjust = 1
	}
	if brigthnessScaler == nil {
		brigthnessScaler = NewLinearBrightnessScaler(0.05, 1)
	}
	if frequencyScaler == nil {
		frequencyScaler = NewLinearFrequencyScaler(.5, 10, .1)
	}

	for id := LedID(0); id < ledsCount; id++ {
		ledSpecs[id].name = LedName(id)
	}
	if err := rpio.Open(); err != nil {
		return err
	}

	running = true
	// Time the loop
	timingChan := make(chan int)
	timingLoops := 3000
	go loop(timingChan, timingLoops)
	loopμs = <-timingChan / timingLoops
	close(timingChan)
	logger.Info("estimed loop duration", "μs", loopμs)
	return nil
}

func Stop() error {
	logger.Info("Pidp.Stop")
	if !running {
		return nil
	}
	running = false
	ClearLeds(0)
	time.Sleep(50 * time.Millisecond) // wait for the loop to notice
	return rpio.Close()
}

func Events() <-chan Event {
	return events
}

func GetBrightnessAdjust() float64 {
	return brightnessAdjust
}

// Sets the global brightness level - eg if in a dark room you could use
// a low value.
func SetBrightnessAdjust(adjust float64) {
	brightnessAdjust = adjust
}

// The Led() function is passed a "logical" brightness param [0,  1].
// If non-zero, this value will be mapped to a "physical" value controlling
// the number of cycles the led will stay on/off.
func SetBrightnessScaler(scaler Scaler) {
	brigthnessScaler = scaler
}

// When the Led() function is given a flashing or strobing effect,
// it is also given a "logical" param [0,  1] indicating the desired
// intensity of the effect.
// The frequency scaler maps this value to a frequency, which is
// hopefully visually meaningful:
//   - The min frequency should not be so low that the led looks off.
//   - The max frequency should not be higher than necessary, as high
//     frequencies are difficult to differentiate visually.
func SetFrequencyScaler(scaler Scaler) {
	frequencyScaler = scaler
}

// Switches off all leds, ramping down brightness for the given duration.
func ClearLeds(offMs int) {
	fx := NewSimpleEffect(0, offMs)
	for id := LedID(0); id < ledsCount; id++ {
		Led(LedID(id), 0, fx)
	}
}

// Sets the led state.
// The brightness is a [0, 1] value.
// The other parameters are interpreted by the effect, which may
// decide to panic if the parameters are invalid.
func Led(id LedID, brightP float64, fx Effect, fxParams ...float64) {
	spec := &ledSpecs[id]
	logger.Debug("Led", "led", spec.name, "brightnessP", brightP, "fx", fx, "fxParams", fxParams)
	brightP = brigthnessScaler.Scale(brightP) * brightnessAdjust
	spec.Lock()
	defer spec.Unlock()
	progress := spec.getProgress()
	spec.makeEnvelope(brightP, fx, fxParams...)
	spec.setProgress(progress)
}

func (spec *ledSpec) isOn(counter int) bool {
	spec.step()
	return brightnessPhases[spec.bright][counter%(brightnessSteps-1)]
}

func loop(timingChan chan int, timingLoops int) {
	// All pins as inputs, pull-ups on columns, pull-offs on rows
	for _, ledrow := range ledRows {
		pin := rpio.Pin(ledrow)
		pin.Input()
		pin.Low()
	}
	for _, col := range gpioCols {
		rpio.Pin(col).Input()
	}
	for _, row := range gpioRows {
		rpio.Pin(row).Input()
	}
	for _, col := range gpioCols {
		rpio.Pin(col).PullUp()
	}
	for _, ledrow := range ledRows {
		rpio.Pin(ledrow).PullOff()
	}
	for _, row := range gpioRows {
		rpio.Pin(row).PullOff()
	}

	// Main loop, exits when .running is false
	counter := 1
	start := time.Now()
	for {
		if counter == timingLoops {
			μs := int(time.Now().Sub(start).Microseconds())
			timingChan <- μs
		}

		// LEDs
		for _, col := range gpioCols {
			rpio.Pin(col).Output()
		}
		for ledrownum, ledrow := range ledRows {
			for colnum, col := range gpioCols {
				led := ledrownum*len(gpioCols) + colnum
				if ledSpecs[led].isOn(counter) {
					rpio.Pin(col).Low()
				} else {
					rpio.Pin(col).High()
				}
			}
			rpio.Pin(ledrow).High()
			rpio.Pin(ledrow).Output()
			nanosleep(5e4) // led is on
			rpio.Pin(ledrow).Low()
			nanosleep(antiGhostingPauseNs)
		}

		// Switches
		for _, col := range gpioCols {
			rpio.Pin(col).Input()
		}
		for rownum, row := range gpioRows {
			rpio.Pin(row).Output()
			rpio.Pin(row).Low()
			nanosleep(500)
			for colnum, col := range gpioCols {
				reading := rpio.Pin(col).Read()
				nid := nativeSwitchID(rownum*len(gpioCols) + colnum)
				oldState := switches[nid]
				newState := reading == rpio.Low
				if nid == swTEST {
					// Have false for rest position
					newState = !newState
				}
				switches[nid] = newState
				if newState != oldState {
					evt := makeEvent(nid, newState)
					if evt.ID != SS_NIL {
						events <- evt
					}
				}
			}
			rpio.Pin(row).Input()
		}

		if !running {
			break
		}
		counter++
	}
}

func makeEvent(nid nativeSwitchID, state bool) Event {
	synEvt := NON_EVENT

	doMomentary := func(id SwitchID) {
		if state {
			synEvt = Event{ID: id, On: true}
		}
	}

	switch nid {
	case swKNOBA_CW, swKNOBA_ACW, swKNOBD_CW, swKNOBD_ACW:
		synEvt = eventForKnob(nid, state)
	case swKNOBA_PUSH:
		doMomentary(SS_KNOBA_PUSH)
	case swKNOBD_PUSH:
		doMomentary(SS_KNOBD_PUSH)
	case swTEST:
		synEvt = Event{ID: SS_TEST, On: state}
	case swLOAD:
		doMomentary(SS_LOAD)
	case swEXAM:
		doMomentary(SS_EXAM)
	case swDEP:
		doMomentary(SS_DEP)
	case swCONT:
		doMomentary(SS_CONT)
	case swENABLE:
		id := SS_ENABLE
		if state {
			id = SS_HALT
		}
		synEvt = Event{ID: id, On: true}
	case swSINST:
		id := SS_S_INST
		if state {
			id = SS_S_BUS_CYCLE
		}
		synEvt = Event{ID: id, On: true}
	case swSTART:
		doMomentary(SS_START)
	}
	// Register switches: as well as emitting an event, we track
	// the position of the switches, see Pidp.ReadRegSwitches()
	if nid >= swSR0 && nid <= swSR21 {
		switches[nid] = state
		synEvt = Event{
			ID: SS_SR0 + SwitchID(nid-swSR0),
			On: state,
		}
	}
	return synEvt
}

// Returns the integer indicated by the register switches.
func ReadRegSwitches() uint {
	val := uint(0)
	for i := 0; i < 22; i++ {
		if switches[SwitchID(i)] {
			val ^= 1 << i
		}
	}
	return val
}

func assert(b bool, format string, args ...any) {
	if !b {
		panic(fmt.Sprintf("assertion failed: "+format, args...))
	}
}
