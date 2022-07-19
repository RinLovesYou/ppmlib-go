package ppmlib

type Layer struct {
	PenColor PenColor

	linesEncoding []byte
	layerData     []byte
}

func NewLayer() *Layer {
	return &Layer{
		linesEncoding: make([]byte, 48),
		layerData:     make([]byte, 32*192),
	}
}

func (l *Layer) LineEncodingAt(index int) LineEncoding {
	return LineEncoding((l.linesEncoding[index>>2] >> ((index & 0x3) << 1)) & 0x3)
}

func (l *Layer) Get(x, y int) bool {
	p := 256*y + x
	return (l.layerData[p>>3] & ((byte)(1 << (p & 7)))) != 0
}

func (l *Layer) Set(x, y int, val bool) {
	p := 256*y + x
	l.layerData[p>>3] &= byte(^(1 << (p & 0x7)))

	toSet := byte(0)
	if val {
		toSet = 1
	}

	l.layerData[p>>3] |= toSet << (p & 0x7)
}

func (l *Layer) SetLineEncoding(lineIndex int, value LineEncoding) {
	o := lineIndex >> 2
	pos := (lineIndex & 0x3) * 2
	b := l.linesEncoding[o]

	b = byte(b & byte(^(0x3 << pos)))
	b |= byte(value << pos)
	l.linesEncoding[o] = b

}

func (l *Layer) ChooseLineEncoding(y int) LineEncoding {
	var chks0, chks1 int
	i := 32 * y

	for b := 0; b < 32; b++ {
		if l.layerData[i] == 0 {
			chks0 += 1
		}

		if l.layerData[i] == 0xFF {
			chks1 += 1
		}
		i++
	}

	if chks0 == 32 {
		return LineEncodingSkip
	}

	if chks0 == 0 && chks1 == 1 {
		return LineEncodingRaw
	}

	if chks0 > chks1 {
		return LineEncodingCoded
	}

	return LineEncodingInvertedCoded
}
