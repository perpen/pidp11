package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/perpen/pidp11"
)

func main() {
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:   slog.LevelInfo,
		NoColor: true,
	}))

	if false {
		// These settings are already set by default.

		pidp11.SetBrightnessAdjust(1)

		// These two next settings probably won't need to be changed
		pidp11.SetBrightnessScaler(pidp11.NewLinearBrightnessScaler(
			.03, // minimum
			1,   // max
		))
		pidp11.SetFrequencyScaler(pidp11.NewLinearFrequencyScaler(
			.5, // minimum frequency
			10, // maximum frequency
			.1, // we'll have 1Hz for this input value
		))
	}

	pidp11.Start(logger)
	defer pidp11.Stop()

	lightshow := func() {
		slog.Info("lightshow")

		// The leds are controlled using function
		//   Led(<led>, <brightness>, <effect>, [<effect param>...])
		// Each call takes only a few microseconds, that is the
		// time for the effect to make the brightness envelope with
		// the given parameters.

		// Switch on immediately
		pidp11.Led(pidp11.LED_A0, 1, pidp11.NewSimpleEffect(0, 0))

		// Switch off with ramping down
		pidp11.Led(pidp11.LED_A2, 1, pidp11.NewSimpleEffect(0, 3000))
		pidp11.Led(pidp11.LED_A2, 0, pidp11.NewSimpleEffect(0, 3000))

		// Switch on with ramping up
		pidp11.Led(pidp11.LED_A4, 1, pidp11.NewSimpleEffect(3000, 0))

		// Some effects require parameters.
		// For example for flashing and strobing we give give a [0, 1] number
		// that will be mapped to a frequency using the frequency scaler.
		// Here we give a low value to this parameter in order to get a low
		// frequency and thus make the brightness changes easier to notice.
		const periodicEffectParam = 0.1

		// Periodic without ramping up/down
		pidp11.Led(pidp11.LED_A6, 1, pidp11.NewFlashEffect(0, 0), periodicEffectParam)
		pidp11.Led(pidp11.LED_A7, 1, pidp11.NewStrobeEffect(0, 0), periodicEffectParam)

		// Periodic with ramping up/down
		pidp11.Led(pidp11.LED_A9, 1, pidp11.NewFlashEffect(250, 250), periodicEffectParam)
		pidp11.Led(pidp11.LED_A10, 1, pidp11.NewStrobeEffect(250, 250), periodicEffectParam)

		// Periodic with ramping up
		pidp11.Led(pidp11.LED_A12, 1, pidp11.NewFlashEffect(500, 0), periodicEffectParam)
		pidp11.Led(pidp11.LED_A13, 1, pidp11.NewStrobeEffect(500, 0), periodicEffectParam)

		// Periodic with ramping down
		pidp11.Led(pidp11.LED_A15, 1, pidp11.NewFlashEffect(0, 500), periodicEffectParam)
		pidp11.Led(pidp11.LED_A16, 1, pidp11.NewStrobeEffect(0, 500), periodicEffectParam)

		// Various brightness levels
		pidp11.Led(pidp11.LED_A18, .1, pidp11.NewSimpleEffect(0, 0))
		pidp11.Led(pidp11.LED_A19, .5, pidp11.NewSimpleEffect(0, 0))
		pidp11.Led(pidp11.LED_A20, 1, pidp11.NewSimpleEffect(0, 0))

		// Error effect
		pidp11.Led(pidp11.LED_PAR_ERR, 1, pidp11.NewErrorEffect())
	}

	slog.Info("Action switches/knobs to trigger events")
	slog.Info("Press START to restart the underwhelming lightshow")
	slog.Info("Press the Data knob to exit")
	lightshow()
	for ev := range pidp11.Events() {
		slog.Info("event loop", "event", ev)
		switch ev.ID {
		case pidp11.SS_KNOBD_PUSH:
			return
		case pidp11.SS_START:
			pidp11.ClearLeds(0)
			time.Sleep(time.Second)
			lightshow()
		default:

		}
	}
}
