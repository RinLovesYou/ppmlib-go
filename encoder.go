package ppmlib

import "math"

func Encode(in []int, out *[]byte) {
	leftStatus := NewStatus()
	rightStatus := NewStatus()

	for i := 0; i < len(in)-1; i += 2 {
		sample := in[i]
		left := leftStatus.Encode(sample)

		sample = in[i+1]
		right := rightStatus.Encode(sample)

		b := left<<4 | right&0x0f
		*out = append(*out, b)
	}
}

type Status struct {
	sample int
	index  int
}

func NewStatus() *Status {
	return &Status{
		sample: 0,
		index:  0,
	}
}

func (status *Status) Decode(nibble byte) int {
	step := stepTable[status.index]

	diff := 0
	if nibble&4 != 0 {
		diff += step
	}
	if nibble&2 != 0 {
		diff += step >> 1
	}
	if nibble&1 != 0 {
		diff += step >> 2
	}
	diff += step >> 3

	if nibble&8 != 0 {
		diff = -diff
	}

	newSample := status.sample + diff
	if newSample > math.MaxInt16 {
		newSample = math.MaxInt16
	} else if newSample < math.MinInt16 {
		newSample = math.MinInt16
	}
	status.sample = newSample

	index := status.index + indexTable[nibble]
	if index < 0 {
		index = 0
	} else if index >= len(stepTable) {
		index = len(stepTable) - 1
	}
	status.index = index

	return newSample
}

func (status *Status) Encode(sample int) byte {
	diff := sample - status.sample
	var nibble byte = 0

	if diff < 0 {
		nibble = 8
		diff = -diff
	}

	var mask byte = 4
	tempStep := stepTable[status.index]
	for i := 0; i < 3; i++ {
		if diff > tempStep {
			nibble |= mask
			diff -= tempStep
		}
		mask >>= 1
		tempStep >>= 1
	}

	// XXX
	// Update the status
	status.Decode(nibble)

	return nibble
}
