package ppmlib

import (
	crunch "github.com/superwhiskers/crunch/v3"
)

type Frame struct {
	FirstByteHeader byte

	translateX int8
	translateY int8

	PaperColor PaperColor

	Layer1 *Layer
	Layer2 *Layer

	AnimationIndex int
}

func NewFrame() *Frame {
	return &Frame{
		Layer1: NewLayer(),
		Layer2: NewLayer(),
	}
}

func ReadFrame(buffer *crunch.Buffer) *Frame {
	frame := NewFrame()

	frame.FirstByteHeader = buffer.ReadByteNext()

	if frame.FirstByteHeader&96 != 0 {
		frame.translateX = int8(buffer.ReadByteNext())
		frame.translateY = int8(buffer.ReadByteNext())
	}

	frame.PaperColor = PaperColor(frame.FirstByteHeader % 2)
	frame.Layer1.PenColor = PenColor((frame.FirstByteHeader >> 1) & 3)
	frame.Layer2.PenColor = PenColor((frame.FirstByteHeader >> 3) & 3)

	frame.Layer1.linesEncoding = buffer.ReadBytesNext(48)
	frame.Layer2.linesEncoding = buffer.ReadBytesNext(48)

	//this is about to get really disgusting. I love line encoding. bare with me.

	for y := 0; y < 192; y++ {
		yy := y << 5

		switch frame.Layer1.LineEncodingAt(y) {
		case LineEncodingCoded:
			for x := 0; x < 32; x++ {
				frame.Layer1.layerData[yy+x] = 0
			}

			b1 := buffer.ReadByteNext()
			b2 := buffer.ReadByteNext()
			b3 := buffer.ReadByteNext()
			b4 := buffer.ReadByteNext()

			bytes := uint32(b1)<<24 + uint32(b2)<<16 + uint32(b3)<<8 + uint32(b4)

			for bytes != 0 {
				if bytes&0x80000000 != 0 {
					frame.Layer1.layerData[yy] = buffer.ReadByteNext()
				}
				bytes <<= 1
				yy++
			}
		case LineEncodingInvertedCoded:
			for x := 0; x < 32; x++ {
				frame.Layer1.layerData[yy+x] = 0xFF
			}

			b1 := buffer.ReadByteNext()
			b2 := buffer.ReadByteNext()
			b3 := buffer.ReadByteNext()
			b4 := buffer.ReadByteNext()

			bytes := uint32(b1)<<24 + uint32(b2)<<16 + uint32(b3)<<8 + uint32(b4)

			for bytes != 0 {
				if bytes&0x80000000 != 0 {
					frame.Layer1.layerData[yy] = buffer.ReadByteNext()
				}
				bytes <<= 1
				yy++
			}

		case LineEncodingRaw:
			for x := 0; x < 32; x++ {
				frame.Layer1.layerData[yy+x] = buffer.ReadByteNext()
			}
		}
	}

	for y := 0; y < 192; y++ {
		yy := y << 5

		switch frame.Layer2.LineEncodingAt(y) {
		case LineEncodingCoded:
			for x := 0; x < 32; x++ {
				frame.Layer2.layerData[yy+x] = 0
			}

			b1 := buffer.ReadByteNext()
			b2 := buffer.ReadByteNext()
			b3 := buffer.ReadByteNext()
			b4 := buffer.ReadByteNext()

			bytes := uint32(b1)<<24 + uint32(b2)<<16 + uint32(b3)<<8 + uint32(b4)

			for bytes != 0 {
				if bytes&0x80000000 != 0 {
					frame.Layer2.layerData[yy] = buffer.ReadByteNext()
				}
				bytes <<= 1
				yy++
			}
		case LineEncodingInvertedCoded:
			for x := 0; x < 32; x++ {
				frame.Layer2.layerData[yy+x] = 0xFF
			}

			b1 := buffer.ReadByteNext()
			b2 := buffer.ReadByteNext()
			b3 := buffer.ReadByteNext()
			b4 := buffer.ReadByteNext()

			bytes := uint32(b1)<<24 + uint32(b2)<<16 + uint32(b3)<<8 + uint32(b4)

			for bytes != 0 {
				if bytes&0x80000000 != 0 {
					frame.Layer2.layerData[yy] = buffer.ReadByteNext()
				}
				bytes <<= 1
				yy++
			}

		case LineEncodingRaw:
			for x := 0; x < 32; x++ {
				frame.Layer2.layerData[yy+x] = buffer.ReadByteNext()
			}
		}
	}

	return frame
}

func (f *Frame) Overwrite(other *Frame) {
	if f.FirstByteHeader&0x80 != 0 {
		return
	}

	var ld0 int
	del := byte(f.translateX & 7)
	pi0 := int(-f.translateX >> 3)

	if f.translateX >= 0 {
		ld0 = int(f.translateX >> 3)
		pi0 = 0
		del = byte(f.translateX) & byte(7)
	}

	ndel := byte(8) - del
	alpha := byte((1 << (byte(8) - del)) - 1)
	nalpha := ^alpha

	var pi, ld int

	if f.translateX >= 0 {
		for y := 0; y < 192; y++ {
			if y < int(f.translateY) {
				continue
			}

			if y-int(f.translateY) >= 192 {
				break
			}

			ld = (y << 5) + ld0
			pi = ((y - int(f.translateY)) << 5) + pi0
			f.Layer1.layerData[ld] ^= other.Layer1.layerData[pi] & alpha
			f.Layer2.layerData[ld] ^= other.Layer2.layerData[pi] & alpha
			ld++

			for (ld & 31) < 31 {
				f.Layer1.layerData[ld] ^= (((other.Layer1.layerData[pi] & nalpha) >> ndel) | ((other.Layer1.layerData[pi+1] & alpha) << del))
				f.Layer2.layerData[ld] ^= (((other.Layer2.layerData[pi] & nalpha) >> ndel) | ((other.Layer2.layerData[pi+1] & alpha) << del))
				ld++
				pi++
			}

			f.Layer1.layerData[ld] ^= ((other.Layer1.layerData[pi] & nalpha) | (other.Layer1.layerData[pi+1] & alpha))
			f.Layer2.layerData[ld] ^= ((other.Layer2.layerData[pi] & nalpha) | (other.Layer2.layerData[pi+1] & alpha))
		}
	} else {
		for y := 0; y < 192; y++ {
			if y < int(f.translateY) {
				continue
			}

			if y-int(f.translateY) >= 192 {
				break
			}

			ld = (y << 5) + ld0
			pi = ((y - int(f.translateY)) << 5) + pi0

			for (pi & 31) < 31 {
				f.Layer1.layerData[ld] ^= (((other.Layer1.layerData[pi] & nalpha) >> ndel) | ((other.Layer1.layerData[pi+1] & alpha) << del))
				f.Layer2.layerData[ld] ^= (((other.Layer2.layerData[pi] & nalpha) >> ndel) | ((other.Layer2.layerData[pi+1] & alpha) << del))
				ld++
				pi++
			}

			f.Layer1.layerData[ld] ^= other.Layer1.layerData[pi] & nalpha
			f.Layer2.layerData[ld] ^= other.Layer2.layerData[pi] & nalpha

		}
	}

}

func (f *Frame) CreateDiff0(prev *Frame) *Frame {
	frame := NewFrame()
	frame.FirstByteHeader = f.FirstByteHeader

	for i := 0; i < 32*192; i++ {
		frame.Layer1.layerData[i] = f.Layer1.layerData[i] ^ prev.Layer1.layerData[i]
		frame.Layer2.layerData[i] = f.Layer2.layerData[i] ^ prev.Layer2.layerData[i]
	}

	for y := 0; y < 192; y++ {
		frame.Layer1.SetLineEncoding(y, frame.Layer1.ChooseLineEncoding(y))
		frame.Layer2.SetLineEncoding(y, frame.Layer1.ChooseLineEncoding(y))
	}

	return frame
}

func (f *Frame) Bytes() []byte {
	res := make([]byte, 0)
	res = append(res, f.FirstByteHeader)
	for l := 0; l < 192; l++ {
		f.Layer1.SetLineEncoding(l, f.Layer1.ChooseLineEncoding(l))
		f.Layer2.SetLineEncoding(l, f.Layer2.ChooseLineEncoding(l))
	}

	res = append(res, f.Layer1.linesEncoding...)
	res = append(res, f.Layer2.linesEncoding...)

	for l := 0; l < 192; l++ {
		res = f.LayerPutLine(l, res, true)
	}
	for l := 0; l < 192; l++ {
		res = f.LayerPutLine(l, res, false)
	}

	return res
}

func (f *Frame) LayerPutLine(y int, list []byte, layer1 bool) []byte {
	layer := f.Layer2
	if layer1 {
		layer = f.Layer1
	}
	compr := int(layer.LineEncodingAt(y))
	if compr == 0 {
		return list
	}

	if compr == 1 {
		chks := make([]byte, 0)
		flag := uint32(0)

		for i := 0; i < 32; i++ {
			chunk := byte(0)
			for j := 0; j < 8; j++ {
				if layer.Get(8*i+j, y) {
					chunk |= 1 << uint(j)
				}
			}

			if chunk != 0x00 {
				flag |= uint32(1) << (31 - i)
				chks = append(chks, chunk)
			}
		}

		list = append(list, byte((flag&0xFF000000)>>24))
		list = append(list, byte((flag&0x00FF0000)>>16))
		list = append(list, byte((flag&0x0000FF00)>>8))
		list = append(list, byte((flag & 0x000000FF)))
		list = append(list, chks...)

		return list
	}

	if compr == 2 {
		chks := make([]byte, 0)
		flag := 0

		for i := 0; i < 32; i++ {
			chunk := byte(0)
			for j := 0; j < 8; j++ {
				if layer.Get(8*i+j, y) {
					chunk |= 1 << uint(j)
				}
			}

			if chunk != 0xFF {
				flag |= 1 << (31 - i)
				chks = append(chks, chunk)
			}
		}

		list = append(list, byte((flag&0xFF000000)>>24))
		list = append(list, byte((flag&0x00FF0000)>>16))
		list = append(list, byte((flag&0x0000FF00)>>8))
		list = append(list, byte((flag & 0x000000FF)))
		list = append(list, chks...)

		return list
	}

	if compr == 3 {
		for i := 0; i < 32; i++ {
			chunk := byte(0)

			for j := 0; j < 8; j++ {
				if layer.Get(8*i+j, y) {
					chunk |= 1 << uint(j)
				}
			}

			list = append(list, chunk)
		}

		return list
	}

	return list
}
