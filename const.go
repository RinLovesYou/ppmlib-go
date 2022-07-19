package ppmlib

type PaperColor byte

const (
	PaperColorBlack PaperColor = iota
	PaperColorWhite
)

func (p PaperColor) String() string {
	switch p {
	case PaperColorBlack:
		return "Black"
	case PaperColorWhite:
		return "White"
	}

	return "Unknown"
}

type PenColor int

const (
	// PenColorInverted is the opposite of the current PaperColor.
	PenColorInverted PenColor = 1
	PenColorRed      PenColor = 2
	PenColorBlue     PenColor = 3
)

func (p PenColor) String() string {
	switch p {
	case PenColorInverted:
		return "Inverted"
	case PenColorRed:
		return "Red"
	case PenColorBlue:
		return "Blue"
	}

	return "Unknown"
}

type LineEncoding int

const (
	LineEncodingSkip LineEncoding = iota
	LineEncodingCoded
	LineEncodingInvertedCoded
	LineEncodingRaw
)

func (l LineEncoding) String() string {
	switch l {
	case LineEncodingSkip:
		return "Skip"
	case LineEncodingCoded:
		return "Coded"
	case LineEncodingInvertedCoded:
		return "InvertedCoded"
	case LineEncodingRaw:
		return "Raw"
	}

	return "Unknown"
}

type PPMAudioTrack byte

const (
	BGM PPMAudioTrack = iota
	SE1
	SE2
	SE3
)

func (p PPMAudioTrack) String() string {
	switch p {
	case BGM:
		return "BGM"
	case SE1:
		return "SE1"
	case SE2:
		return "SE2"
	case SE3:
		return "SE3"
	}

	return "Unknown"
}
