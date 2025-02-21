Go module for controlling the
[PiDP-11](https://obsolescence.wixsite.com/obsolescence/pidp-11).

# Light effects

A number of effects are supported:
- Simple (one-shot): optional ramping up or down of brightness when
  switching a led on or off.
- Flash (periodic): the light stays on and off for the same duration.
  Supports ramping up/down.
- Strobe (periodic): the light stays on for always the same short duration,
  and off for a varying duration.
  Supports ramping up/down.
- Error (periodic): creates a hopefully recognisable brightness envelope.

# Envelopes

New effects can easily be added by making new implementations of the
`Effect` interface.

Effects are implemented using an envelope (as in audio synthesis) which
defines brightness changes and is either one-shot (eg simply switching a
led on or off) or periodic (flashing etc).

An `envelope` is a sequence of `stage`s. A stage defines a linear progression
between two brightness levels, over a certain duration.

# Events

When a switch is actioned, an event is emitted on buffered channel `Events()`.
The events should be read reasonably quickly to avoid blocking the main loop.

# Example

See `demo/main.go`.
