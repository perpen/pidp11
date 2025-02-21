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

var loopμs int // approx. duration of a loop, for converting durations to loops
var logger *slog.Logger

type Pidp struct {
	Events   <-chan Event
	events   chan Event
	running  bool
	ledSpecs [72]ledSpec
	switches [38]bool // current state per nativeSwitchID
}

// Current brightness of the led, and envelope
type ledSpec struct {
	sync.Mutex
	bright int // brightness, 0-31
	env    envelope
	name   string // for debug messages
}

func NewPidp(logger0 *slog.Logger) *Pidp {
	logger = logger0
	pidp := Pidp{
		events: make(chan Event, 0),
	}
	pidp.Events = pidp.events
	for id := LedID(0); id < LEDS_COUNT; id++ {
		pidp.ledSpecs[id].name = LedName(id)
	}
	return &pidp
}

func (pidp *Pidp) Start() error {
	if err := rpio.Open(); err != nil {
		return err
	}
	pidp.running = true
	go func() {
		// Time the loop
		timingChan := make(chan int)
		timingLoops := 3000
		go pidp.loop(timingChan, timingLoops)
		loopμs = <-timingChan / timingLoops
		close(timingChan)
		logger.Info("estimed loop duration", "μs", loopμs)
	}()
	return nil
}

func (pidp *Pidp) Stop() error {
	logger.Info("Pidp.Stop")
	pidp.running = false
	time.Sleep(20 * time.Millisecond) // wait for the loop to notice
	pidp.ClearLeds()
	return rpio.Close()
}

func (pidp *Pidp) ClearLeds() {
	fx := NewSimpleEffect(0, 0)
	for id := LedID(0); id < LEDS_COUNT; id++ {
		pidp.SetLed(LedID(id), fx, 0)
	}
}

func (pidp *Pidp) SetLed(id LedID, fx Effect, brightP float64, params ...float64) {
	spec := &pidp.ledSpecs[id]
	logger.Debug("SetLed", "led", spec.name, "fx", fx, "brightnessP", brightP, "params", params)
	spec.Lock()
	defer spec.Unlock()
	progress := spec.getProgress()
	spec.makeEnvelope(fx, brightP, params...)
	spec.setProgress(progress)
}

func (spec *ledSpec) isOn(counter int) bool {
	spec.step()
	return BRIGHTNESS_PHASES[spec.bright][counter%(BRIGHTNESS_STEPS-1)]
}

func (pidp *Pidp) loop(timingChan chan int, timingLoops int) {
	// All pins as inputs, pull-ups on columns, pull-offs on rows
	for _, ledrow := range LED_ROWS {
		pin := rpio.Pin(ledrow)
		pin.Input()
		pin.Low()
	}
	for _, col := range COLS {
		rpio.Pin(col).Input()
	}
	for _, row := range ROWS {
		rpio.Pin(row).Input()
	}
	for _, col := range COLS {
		rpio.Pin(col).PullUp()
	}
	for _, ledrow := range LED_ROWS {
		rpio.Pin(ledrow).PullOff()
	}
	for _, row := range ROWS {
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
		for _, col := range COLS {
			rpio.Pin(col).Output()
		}
		for ledrownum, ledrow := range LED_ROWS {
			for colnum, col := range COLS {
				led := ledrownum*len(COLS) + colnum
				if pidp.ledSpecs[led].isOn(counter) {
					rpio.Pin(col).Low()
				} else {
					rpio.Pin(col).High()
				}
			}
			rpio.Pin(ledrow).High()
			rpio.Pin(ledrow).Output()
			nanosleep(5e4) // led is on
			rpio.Pin(ledrow).Low()
			nanosleep(ANTI_GHOSTING_PAUSE_NS)
		}

		// Switches
		for _, col := range COLS {
			rpio.Pin(col).Input()
		}
		for rownum, row := range ROWS {
			rpio.Pin(row).Output()
			rpio.Pin(row).Low()
			nanosleep(500)
			for colnum, col := range COLS {
				reading := rpio.Pin(col).Read()
				nid := nativeSwitchID(rownum*len(COLS) + colnum)
				oldState := pidp.switches[nid]
				newState := reading == rpio.Low
				if nid == SW_TEST {
					// Have false for rest position
					newState = !newState
				}
				pidp.switches[nid] = newState
				if newState != oldState {
					evt := pidp.makeEvent(nid, newState)
					if evt.ID != SS_NIL {
						pidp.events <- evt
					}
				}
			}
			rpio.Pin(row).Input()
		}

		if !pidp.running {
			break
		}
		counter++
	}
}

func (pidp *Pidp) makeEvent(nid nativeSwitchID, state bool) Event {
	synEvt := NON_EVENT

	doMomentary := func(id SwitchID) {
		if state {
			synEvt = Event{ID: id, On: true}
		}
	}

	switch nid {
	case SW_KNOBA_CW, SW_KNOBA_ACW, SW_KNOBD_CW, SW_KNOBD_ACW:
		synEvt = eventForKnob(nid, state)
	case SW_KNOBA_PUSH:
		doMomentary(SS_KNOBA_PUSH)
	case SW_KNOBD_PUSH:
		doMomentary(SS_KNOBD_PUSH)
	case SW_TEST:
		synEvt = Event{ID: SS_TEST, On: state}
	case SW_LOAD:
		doMomentary(SS_LOAD)
	case SW_EXAM:
		doMomentary(SS_EXAM)
	case SW_DEP:
		doMomentary(SS_DEP)
	case SW_CONT:
		doMomentary(SS_CONT)
	case SW_ENABLE:
		id := SS_ENABLE
		if state {
			id = SS_HALT
		}
		synEvt = Event{ID: id, On: true}
	case SW_SINST:
		id := SS_S_INST
		if state {
			id = SS_S_BUS_CYCLE
		}
		synEvt = Event{ID: id, On: true}
	case SW_START:
		doMomentary(SS_START)
	}
	// Register switches: as well as emitting an event, we track
	// the position of the switches, see Pidp.ReadRegSwitches()
	if nid >= SW_SR0 && nid <= SW_SR21 {
		pidp.switches[nid] = state
		synEvt = Event{
			ID: SS_SR0 + SwitchID(nid-SW_SR0),
			On: state,
		}
	}
	return synEvt
}

// Returns the integer indicated by the register switches.
func (pidp *Pidp) ReadRegSwitches() uint {
	val := uint(0)
	for i := 0; i < 22; i++ {
		if pidp.switches[SwitchID(i)] {
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
