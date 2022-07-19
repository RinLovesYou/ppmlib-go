package ppmlib

import (
	"math"

	"github.com/RinLovesYou/ppmlib-go/utils"
)

func Encode(in []int, out *[]byte) {
	encoder := newEncoder()
	encoder.encode(in, out)
}

type encoder struct {
	prevSample int
	stepIndex  int
}

func newEncoder() *encoder {
	return &encoder{}
}

func (e *encoder) encode(in []int, out *[]byte) {
	for i := 0; i < len(in)-1; i += 2 {
		encSample := e.encodeSample(in[i])
		encSample2 := e.encodeSample(in[i+1]) << 4
		*out = append(*out, byte(encSample|encSample2))
	}
}

func (e *encoder) encodeSample(sample int) int {
	delta := sample - e.prevSample
	encSample := 0

	if delta < 0 {
		encSample = 8
		delta = -delta
	}

	encSample += int(math.Min(7, float64(delta*4/stepTable[e.stepIndex])))
	e.prevSample = sample
	e.stepIndex = utils.Clamp(e.stepIndex+indexTable[encSample], 0, 80)
	return encSample
}
