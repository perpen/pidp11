Go module for controlling the
[PiDP-11](https://obsolescence.wixsite.com/obsolescence/pidp-11), ie
activating leds and getting switch events.

[Documentation](https://pkg.go.dev/github.com/perpen/pidp11)

Used by [blink11](https://github.com/perpen/blink11/).

# Light effects

A number of effects are supported:
- Simple (one-shot): optional ramping up or down of brightness when
  switching a led on or off.
- Flash (periodic): the light stays on and off for the same duration.
  Supports ramping up/down.
- Strobe (periodic): the light stays on for always the same short
  duration, and off for a varying duration. Supports ramping up/down.
- Error (periodic): creates a hopefully recognisable brightness envelope.

# Brightness envelopes

New effects can easily be added by making new implementations of the
`Effect` interface.

Effects are implemented using an envelope (as in audio synthesis) which
defines brightness changes and is either one-shot (eg simply switching
a led on or off) or periodic (flashing etc).

An `envelope` is a sequence of `stage`s. A stage defines a linear
progression between two brightness levels, over a certain duration.

## Brightness transitions

When switching a led on or off, the current brightness of the led is
taken into account. For example if the led brightness is at 50% and we
switch it off with a ramping down duration of 1 second, the led will
be off after half a second.

Similarly, when changing the effect for a led (eg changing from flashing
to strobing, changing the flashing frequency, changing the ramping up/down
durations or any combinations of those), we transition smoothly to avoid
jarring visual artifacts.

# Events

When a switch is actioned, an event is emitted on buffered channel
`Events()`. The events should be read reasonably quickly to avoid
blocking the main loop.

# Demo program

See `cmd/demo/main.go`.

Depending on your rpi architecture, build with:
- `GOOS=linux GOARCH=arm GOARM=7 go build ./cmd/demo/`
- `GOOS=linux GOARCH=arm64 go build ./cmd/demo/`
