package ppmlib

import (
	"errors"
	"math"

	"github.com/RinLovesYou/ppmlib-go/utils"
)

type AdpcmDecoder struct {
	stepIndex int
	flipnote  *PPMFile
}

func NewAudioDecoder(flipnote *PPMFile) *AdpcmDecoder {
	return &AdpcmDecoder{
		flipnote: flipnote,
	}
}

var indexTable = [16]int{
	-1, -1, -1, -1, 2, 4, 6, 8,
	-1, -1, -1, -1, 2, 4, 6, 8,
}

var stepTable = []int{
	7, 8, 9, 10, 11, 12, 13, 14, 16, 17,
	19, 21, 23, 25, 28, 31, 34, 37, 41, 45,
	50, 55, 60, 66, 73, 80, 88, 97, 107, 118,
	130, 143, 157, 173, 190, 209, 230, 253, 279, 307,
	337, 371, 408, 449, 494, 544, 598, 658, 724, 796,
	876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066,
	2272, 2499, 2749, 3024, 3327, 3660, 4026, 4428, 4871, 5358,
	5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899,
	15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767, 0,
}

func (d *AdpcmDecoder) GetAudioMasterPcm(dstFreq int) ([]int16, error) {
	var err error

	duration := d.getDuration()

	dstSize := int(duration*float32(dstFreq)) + 1
	master := make([]int16, dstSize+1)

	hasBgm := d.flipnote.Audio.Header.BGMTrackSize > 0
	hasSe1 := d.flipnote.Audio.Header.SE1TrackSize > 0
	hasSe2 := d.flipnote.Audio.Header.SE2TrackSize > 0
	hasSe3 := d.flipnote.Audio.Header.SE3TrackSize > 0

	if hasBgm {
		bgmPcm, err := d.getAudioTrackPcm(dstFreq, BGM)
		if err != nil {
			return nil, err
		}
		master = pcmAudioMix(bgmPcm, master, 0)
	}

	if hasSe1 || hasSe2 || hasSe3 {
		samplesPerFrame := float32(dstFreq) / d.flipnote.Framerate
		var se1Pcm []int16
		var se2Pcm []int16
		var se3Pcm []int16

		if hasSe1 {
			se1Pcm, err = d.getAudioTrackPcm(dstFreq, SE1)
			if err != nil {
				return nil, err
			}
		}

		if hasSe2 {
			se2Pcm, err = d.getAudioTrackPcm(dstFreq, SE2)
			if err != nil {
				return nil, err
			}
		}

		if hasSe3 {
			se3Pcm, err = d.getAudioTrackPcm(dstFreq, SE3)
			if err != nil {
				return nil, err
			}
		}

		seFlags := d.flipnote.SoundEffectFlags

		for i := 0; i < int(d.flipnote.FrameCount); i++ {
			seOffset := int(math.Ceil(float64(i) * float64(samplesPerFrame)))
			flag := seFlags[i]

			if hasSe1 && flag == 1 {
				master = pcmAudioMix(se1Pcm, master, seOffset)
			}

			if hasSe2 && flag == 2 {
				master = pcmAudioMix(se2Pcm, master, seOffset)
			}

			if hasSe3 && flag == 4 {
				master = pcmAudioMix(se3Pcm, master, seOffset)
			}
		}
	}

	return master, nil
}

func (d *AdpcmDecoder) getAudioTrackPcm(dstFreq int, track PPMAudioTrack) ([]int16, error) {
	srcPcm, err := d.decode(track)
	if err != nil {
		return nil, err
	}

	speed := d.flipnote.BGMRate
	framerate := d.flipnote.Framerate

	srcFreq := 8192

	if track == BGM {
		bgmAdjust := (1.0 / speed) / (1.0 / framerate)
		srcFreq = int(float32(srcFreq) * bgmAdjust)
	}

	if int(srcFreq) != dstFreq {
		return pcmResampleNearestNeighbour(srcPcm, srcFreq, dstFreq)
	}

	return srcPcm, nil
}

func pcmResampleNearestNeighbour(src []int16, srcFreq int, dstFreq int) ([]int16, error) {
	srcLen := len(src)
	srcDuration := srcLen / int(srcFreq)
	dstLen := srcDuration * dstFreq
	dst := make([]int16, int(dstLen))
	adjFreq := float32(srcFreq) / float32(dstFreq)

	for dstPtr := 0; dstPtr < int(dstLen); dstPtr++ {
		val := float32(dstPtr) * adjFreq
		dst[dstPtr] = pcmGetSample(src, srcLen, int(val))
	}

	return dst, nil
}

func pcmGetSample(src []int16, srcLen int, srcPtr int) int16 {
	if srcPtr < 0 || srcPtr >= srcLen {
		return 0
	}

	return src[srcPtr]
}

func pcmAudioMix(src []int16, dst []int16, offset int) []int16 {
	srcSize := len(src)
	dstSize := len(dst)

	for i := 0; i < srcSize; i++ {
		if offset+i > dstSize {
			break
		}

		if offset+i >= len(dst) || i >= len(src) {
			break
		}
		samp := dst[offset+i] + (src[i] / 2)
		dst[offset+i] = utils.Clamp(samp, -32768, 32767)
	}

	return dst
}

func (d *AdpcmDecoder) decode(track PPMAudioTrack) ([]int16, error) {
	data := d.flipnote.Audio.Data

	var src []byte
	switch track {
	case BGM:
		src = data.RawBGM
	case SE1:
		src = data.RawSE1
	case SE2:
		src = data.RawSE2
	case SE3:
		src = data.RawSE3
	default:
		return nil, errors.New("invalid track")
	}

	srcSize := len(src)
	dst := make([]int16, srcSize*2)

	var srcPtr, dstPtr, sample, predictor int
	d.stepIndex = 0
	lowNibble := true

	for srcPtr < srcSize {
		sample = int(src[srcPtr] & 0xF)

		if !lowNibble {
			sample = int(src[srcPtr] >> 4)
			srcPtr++
		}
		lowNibble = !lowNibble

		step := stepTable[d.stepIndex]
		diff := step >> 3

		if (sample & 1) != 0 {
			diff += step >> 2
		}
		if (sample & 2) != 0 {
			diff += step >> 1
		}
		if (sample & 4) != 0 {
			diff += step
		}
		if (sample & 8) != 0 {
			diff = -diff
		}

		predictor += diff
		predictor = utils.Clamp(predictor, -32768, 32767)

		d.stepIndex += indexTable[sample]
		d.stepIndex = utils.Clamp(d.stepIndex, 0, 88)
		dst[dstPtr] = int16(predictor)
		dstPtr++
	}

	return dst, nil
}

func (d *AdpcmDecoder) getDuration() float32 {
	return (float32(d.flipnote.FrameCount*100) * float32(1/d.flipnote.Framerate)) / 100
}
