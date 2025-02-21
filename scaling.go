package pidp11

import "fmt"

type Scaler interface {
	Scale(float64) float64
}

type linearBrightnessScaler struct {
	min, max float64
}

func NewLinearBrightnessScaler(min, max float64) Scaler {
	if min < 0 || min > 1 || max < min || max > 1 {
		panic(fmt.Errorf("invalid values: min=%f max=%f", min, max))
	}
	return linearBrightnessScaler{
		min: min,
		max: max,
	}
}

func (scaler linearBrightnessScaler) Scale(bright float64) float64 {
	if bright == 0 {
		return 0
	}
	return scaler.min + bright*(scaler.max-scaler.min)
}

type linearFrequencyScaler struct {
	oneHzAt, minHz, maxHz float64
}

func NewLinearFrequencyScaler(minHz, maxHz, oneHzAt float64) Scaler {
	return linearFrequencyScaler{
		minHz:   minHz,
		maxHz:   maxHz,
		oneHzAt: oneHzAt,
	}
}

func (scaler linearFrequencyScaler) Scale(pct float64) float64 {
	// we want: a + b*oneHzAt = 1 and a + b = maxHz
	a := (1 - scaler.maxHz*scaler.oneHzAt) / (1 - scaler.oneHzAt)
	b := scaler.maxHz - a
	hz := a + b*pct
	if hz < scaler.minHz {
		return 0
	}
	return hz
}
