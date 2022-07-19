package ppmlib

import (
	"io"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

type SoundHeader struct {
	BGMTrackSize           uint32
	SE1TrackSize           uint32
	SE2TrackSize           uint32
	SE3TrackSize           uint32
	CurrentFrameSpeed      byte
	RecordingBGMFrameSpeed byte
}

func NewSoundHeader() *SoundHeader {
	return &SoundHeader{}
}

type SoundData struct {
	RawBGM []byte
	RawSE1 []byte
	RawSE2 []byte
	RawSE3 []byte
}

func NewSoundData() *SoundData {
	return &SoundData{
		RawBGM: make([]byte, 0),
		RawSE1: make([]byte, 0),
		RawSE2: make([]byte, 0),
		RawSE3: make([]byte, 0),
	}
}

type Audio struct {
	Header *SoundHeader
	Data   *SoundData
}

func NewAudio() *Audio {
	return &Audio{
		Header: NewSoundHeader(),
		Data:   NewSoundData(),
	}
}

func (a *Audio) Export(reader io.WriteSeeker, flipnote *PPMFile, sampleRate int) error {
	decoder := NewAudioDecoder(flipnote)

	if sampleRate == 0 {
		sampleRate = 32768
	}

	decoded, err := decoder.GetAudioMasterPcm(sampleRate)
	if err != nil {
		return err
	}

	intBuffer := make([]int, len(decoded))
	for i, v := range decoded {
		intBuffer[i] = int(v)
	}

	aa := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  sampleRate,
		},
		Data:           intBuffer,
		SourceBitDepth: 16,
	}

	e := wav.NewEncoder(reader, sampleRate, 16, 1, 1)

	if err := e.Write(aa); err != nil {
		return err
	}

	if err := e.Close(); err != nil {
		return err
	}

	return nil
}
