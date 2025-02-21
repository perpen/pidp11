A Go library for the PiDP-11

## Light effects

A number of effects are supported:
- Simple (one-shot): optional ramping up or down of brightness when
  switching a led on or off.
- Flash (periodic): the light stays on and off for the same duration (but
  supports ramping up/down)
- Strobe (periodic): the light stays on for always the same duration, and
  off for a varying duration.
- Error (periodic): creates a hopefully recognisable brightness envelope.

### Envelopes

New effects can easily be added by making new implementations of the
`Effect` interface.

Effects are implemented using an envelope (as in audio synthesis), which
is either one-shot (eg simply switching a led on or off) or periodic (flashing etc).

An `envelope` is a sequence of `stage`s. A stage defines a linear progression
between two brightness levels, over a certain duration.

## Events

When a switch is actioned, an event is emitted on buffered channel `Pidp.Events`.
The events should be read reasonably quickly to avoid blocking the main loop.

## API

xxx provide sample command for testing?

```
pidp.Start()
defer pidp.Stop()

brightnessMin := 0
brigthnessMax := 1
oneHzAt := .1
minHz := 1
maxHz := 10

func scaleBrightness(bright float64) float64 {
	if bright == 0 {
		return 0
	}
	return brightnessMin + bright*brightnessScaling*(brightnessMax-brightnessMin)
}

func percentToHz(pct float64) float64 {
	// we want: a + b*oneHzAt = 1 and a + b = maxHz
	a := (1 - maxHz*oneHzAt) / (1 - oneHzAt)
	b := maxHz - a
	hz := a + b*pct
	if hz < minHz {
		return 0
	}
	return hz
}

basicFx := pidp.NewSimpleEffect(0, 0)
func basicLed(id LedID, on bool) {
	pidp.SetLed(id, basicFx, on)
}

pidp.SetLed(LED_A0, pidp.NewSimpleEffect(0, 0), 1)
pidp.SetLed(LED_A1, pidp.NewSimpleEffect(500, 1000), 1)
pidp.SetLed(LED_A2, pidp.NewSimpleEffect(500, 1000), 1)
pidp.SetLed(LED_A3, pidp11.NewFlashEffect(200, 200, percentToHz), .5)
pidp.SetLed(LED_A4, pidp11.NewStrobeEffect(200, 200, percentToHz), .5)
pidp.SetLed(LED_A5, pidp11.NewErrorEffect(), 1)

for ev := range pidp.Events {
	fmt.Printf("%s %s %v\n", ev, pidp.SwitchName(ev.ID), ev.State)
}
```
