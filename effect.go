package pidp11

import (
	"math"
)

type Effect interface {
	makeEnvelope(spec *ledSpec, bright int, fxParams ...float64)
}

// One-shot attack or release, onMs used when switching on, offMs when off.
// No parameters when passed to Led().
type SimpleEffect struct {
	onMs, offMs int
}

func NewSimpleEffect(onMs, offMs int) SimpleEffect {
	return SimpleEffect{
		onMs:  onMs,
		offMs: offMs,
	}
}

func (fx SimpleEffect) makeEnvelope(spec *ledSpec, bright int, fxParams ...float64) {
	assertParams(0, fxParams)
	env := &spec.env
	delta := abs(bright - spec.bright)
	var fxMs int
	if bright == 0 {
		fxMs = fx.offMs
	} else {
		fxMs = fx.onMs
	}
	ms := scale(delta, 31, 0, fxMs)
	env.addStage(spec.bright, bright, ms, true)
}

// Periodic strobing, the led stays on for a fixed amount of time
// but the duration of the off-time is variable.
// When passed to Led(), requires a [0, 1] parameter which will be
// mapped to a frequency using the function set by SetFrequencyScaler().
type StrobeEffect struct {
	strobeOnLoops int // number of loops the led will stay on
	onMs, offMs   int
	// xxx expose param, or remove:
	lowDivider int // used to compute the low value from the requested brightness
}

func NewStrobeEffect(onMs, offMs int) StrobeEffect {
	return StrobeEffect{
		strobeOnLoops: 60,
		onMs:          onMs,
		offMs:         offMs,
		lowDivider:    999,
	}
}

func (fx StrobeEffect) makeEnvelope(spec *ledSpec, bright int, fxParams ...float64) {
	assertParams(1, fxParams)
	hz := frequencyScaler.Scale(fxParams[0])
	onMs := fx.onMs
	offMs := fx.offMs
	if hz == 0 {
		spec.env.addStage(spec.bright, 0, offMs, true)
		return
	}
	periodMs := int(math.Round(1e3 / hz))
	strobeOnMs := loopμs * fx.strobeOnLoops / 1e3
	restMs := periodMs - strobeOnMs
	assert(restMs >= 0, "restMs=%d", restMs)
	if onMs+offMs > restMs {
		shrinkage := float64(restMs) / float64(onMs+offMs)
		onMs = int(math.Floor(float64(onMs) * shrinkage))
		offMs = int(math.Floor(float64(offMs) * shrinkage))
		assert(onMs+offMs <= restMs, "onMs=%d, offMs=%d, restMs=%d",
			onMs, offMs, restMs)
	}
	upMs := onMs + strobeOnMs
	downMs := periodMs - upMs
	spec.setupASRS(bright, bright/fx.lowDivider, onMs, offMs, upMs, downMs)
}

// Periodic flashing, the led stays on and off for the same amount of time.
// When passed to Led(), requires a [0, 1] parameter which will be
// mapped to a frequency using the function set by SetFrequencyScaler().
type FlashEffect struct {
	onMs, offMs int
	lowDivider  int // used to compute the low value from the requested brightness
}

func NewFlashEffect(onMs, offMs int) FlashEffect {
	return FlashEffect{
		onMs:       onMs,
		offMs:      offMs,
		lowDivider: 999,
	}
}

func (fx FlashEffect) makeEnvelope(spec *ledSpec, bright int, fxParams ...float64) {
	assertParams(1, fxParams)
	hz := frequencyScaler.Scale(fxParams[0])
	onMs := fx.onMs
	offMs := fx.offMs
	if hz == 0 {
		spec.env.addStage(spec.bright, 0, offMs, true)
		return
	}
	periodMs := int(math.Round(1e3 / hz))
	assert(periodMs >= 2, "periodMs=%d", periodMs)
	downMs := periodMs / 2
	upMs := downMs
	if offMs > downMs {
		offMs = downMs
	}
	if onMs > upMs {
		onMs = upMs
	}
	spec.setupASRS(bright, bright/fx.lowDivider, onMs, offMs, upMs, downMs)
}

// Create an attack-sustain-release-sustain envelope
func (spec *ledSpec) setupASRS(hi, lo, onMs, offMs, upMs, downMs int) {
	logger.Debug("setupASRS", "led", spec.name, "onMs", onMs, "offMs", offMs, "upMs", upMs, "downMs", downMs)
	env := &spec.env
	if onMs > 0 {
		env.addStage(lo, hi, onMs, false) // attack
	}
	env.addStage(hi, hi, upMs-onMs, false) // sustain high
	if offMs > 0 {
		env.addStage(hi, lo, offMs, false) // release
	}
	env.addStage(lo, lo, downMs-offMs, false) //sustain low
}

// Periodic, recognisable pulsating envelope.
// Takes no parameters when passed to Led().
type ErrorEffect struct{}

func NewErrorEffect() ErrorEffect {
	return ErrorEffect{}
}

func (fx ErrorEffect) makeEnvelope(spec *ledSpec, bright int, fxParams ...float64) {
	assertParams(0, fxParams)
	hi := bright
	ms := 200
	lo := hi / 4
	spec.env.addStage(hi, 0, ms, false)
	spec.env.addStage(0, lo, ms, false)
	spec.env.addStage(lo, lo, ms, false)
}

func assertParams(count int, params []float64) {
	assert(len(params) == count,
		"expected %d params, got %d", count, len(params))
}

// Linear scaling between input and output ranges
func scale(inVal, inMax, outMin, outMax int) int {
	if inVal == 0 && inMax == 0 || outMin == outMax {
		return outMin
	}
	inFrac := float64(inVal) / float64(inMax)
	return int(math.Round(float64(outMin) + inFrac*float64(outMax-outMin)))
}
