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
			.05, // minimum
			1,   // max
		))
		pidp11.SetFrequencyScaler(pidp11.NewLinearFrequencyScaler(
			.5, // minimum frequency
			10, // maximum frequency
			.1, // we'll have 1Hz for this metric value
		))
	}

	pidp11.Start(logger)
	defer pidp11.Stop()

	// Corresponds to a low strobing/flashing frequency, to make the effects
	// easier to notice.
	const periodicEffectParam = 0.06

	demo := func() {
		slog.Info("lightshow")

		// Switch on immediately
		pidp11.Led(pidp11.LED_A0, pidp11.NewSimpleEffect(0, 0), 1)

		// Switch off with ramping down
		pidp11.Led(pidp11.LED_A1, pidp11.NewSimpleEffect(0, 0), 1)
		pidp11.Led(pidp11.LED_A1, pidp11.NewSimpleEffect(0, 3000), 0)

		// Switch on with ramping up
		pidp11.Led(pidp11.LED_A2, pidp11.NewSimpleEffect(3000, 0), 1)

		// Periodic without ramping up/down
		pidp11.Led(pidp11.LED_A3, pidp11.NewFlashEffect(0, 0), 1, periodicEffectParam)
		pidp11.Led(pidp11.LED_A4, pidp11.NewStrobeEffect(0, 0), 1, periodicEffectParam)

		// Periodic with ramping up/down
		pidp11.Led(pidp11.LED_A5, pidp11.NewFlashEffect(1000, 1000), 1, periodicEffectParam)
		pidp11.Led(pidp11.LED_A6, pidp11.NewStrobeEffect(1000, 1000), 1, periodicEffectParam)

		// Periodic with ramping up
		pidp11.Led(pidp11.LED_A7, pidp11.NewFlashEffect(1000, 0), 1, periodicEffectParam)
		pidp11.Led(pidp11.LED_A8, pidp11.NewStrobeEffect(1000, 0), 1, periodicEffectParam)

		// Periodic with ramping down
		pidp11.Led(pidp11.LED_A9, pidp11.NewFlashEffect(0, 1000), 1, periodicEffectParam)
		pidp11.Led(pidp11.LED_A10, pidp11.NewStrobeEffect(0, 1000), 1, periodicEffectParam)

		// Various brightness levels
		pidp11.Led(pidp11.LED_A11, pidp11.NewSimpleEffect(0, 0), .1)
		pidp11.Led(pidp11.LED_A12, pidp11.NewSimpleEffect(0, 0), .5)
		pidp11.Led(pidp11.LED_A13, pidp11.NewSimpleEffect(0, 0), 1)

		// Error effect
		pidp11.Led(pidp11.LED_A14, pidp11.NewErrorEffect(), 1)
	}

	slog.Info("Action switches/knobs to trigger events")
	slog.Info("Press START to restart the underwhelming lightshow")
	slog.Info("Press the Data knob to exit")
	demo()
	for ev := range pidp11.Events() {
		slog.Info("event loop", "event", ev)
		switch ev.ID {
		case pidp11.SS_KNOBD_PUSH:
			return
		case pidp11.SS_START:
			pidp11.ClearLeds(0)
			time.Sleep(time.Second)
			demo()
		default:

		}
	}
}
