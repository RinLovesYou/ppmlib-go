package ppmlib

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/RinLovesYou/ppmlib-go/utils"
	crunch "github.com/superwhiskers/crunch/v3"
)

type PPMFile struct {
	AnimationDataSize   uint32
	SoundDataSize       uint32
	FrameCount          uint16
	FormatVersion       uint16
	Locked              bool
	ThumbnailFrameIndex uint16

	ParentFilename  *Filename
	CurrentFilename *Filename

	RootAuthor    *Author
	ParentAuthor  *Author
	CurrentAuthor *Author

	RootFileFragment []byte //len: 8

	Timestamp *Timestamp

	Thumbnail []byte `json:"-"` //len: 1536

	FrameOffsetTableSize uint16

	Frames       []*Frame `json:"-"`
	FramesParsed uint16

	AnimationFlags uint16

	Audio *Audio `json:"-"`

	Framerate        float32
	BGMRate          float32
	SoundEffectFlags []byte
	Signature        []byte

	Key *rsa.PrivateKey
}

func ReadFile(path string) (*PPMFile, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	file, err := Parse(bytes)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func Parse(data []byte) (*PPMFile, error) {
	buffer := crunch.NewBuffer(data)
	var err error

	magic := buffer.ReadBytesNext(4)

	if string(magic) != "PARA" {
		return nil, errors.New("invalid file magic")
	}

	file := &PPMFile{}

	file.AnimationDataSize = buffer.ReadU32LENext(1)[0]
	file.SoundDataSize = buffer.ReadU32LENext(1)[0]
	file.FrameCount = buffer.ReadU16LENext(1)[0]

	file.FrameCount++
	file.Frames = make([]*Frame, file.FrameCount)

	file.FormatVersion = buffer.ReadU16LENext(1)[0]

	isLocked := buffer.ReadU16LENext(1)[0]
	file.Locked = isLocked != 0

	file.ThumbnailFrameIndex = buffer.ReadU16LENext(1)[0]
	rootName := utils.ReadUTF16String(buffer.ReadBytesNext(22))
	parentName := utils.ReadUTF16String(buffer.ReadBytesNext(22))
	currentName := utils.ReadUTF16String(buffer.ReadBytesNext(22))

	parentId := buffer.ReadU64LENext(1)[0]
	currentId := buffer.ReadU64LENext(1)[0]

	file.ParentFilename, err = FilenameFrom(buffer.ReadBytesNext(18))
	if err != nil {
		return nil, err
	}

	file.CurrentFilename, err = FilenameFrom(buffer.ReadBytesNext(18))
	if err != nil {
		return nil, err
	}

	rootId := buffer.ReadU64LENext(1)[0]

	file.RootAuthor, err = NewAuthor(rootName, rootId)
	if err != nil {
		return nil, err
	}

	file.ParentAuthor, err = NewAuthor(parentName, parentId)
	if err != nil {
		return nil, err
	}

	file.CurrentAuthor, err = NewAuthor(currentName, currentId)
	if err != nil {
		return nil, err
	}

	file.RootFileFragment = buffer.ReadBytesNext(8)

	file.Timestamp = NewTimestamp(buffer.ReadU32LENext(1)[0])

	buffer.SeekByte(2, true)

	file.Thumbnail = buffer.ReadBytesNext(1536)

	buffer.SeekByte(0x06A0, false)
	file.FrameOffsetTableSize = buffer.ReadU16LENext(1)[0]

	buffer.SeekByte(4, true)

	file.AnimationFlags = buffer.ReadU16LENext(1)[0]

	oCnt := file.FrameOffsetTableSize/4 - 1
	animationOffsets := buffer.ReadU32LENext(int64(oCnt + 1))

	go file.ParseFrames(data, animationOffsets)

	if file.SoundDataSize == 0 {
		return file, nil
	}

	offset := 0x6A0 + int64(file.AnimationDataSize)
	buffer.SeekByte(offset, false)

	file.SoundEffectFlags = buffer.ReadBytesNext(int64(len(file.Frames)))

	file.Audio = NewAudio()

	offset += int64(len(file.Frames))

	//makes the next offset divisible by 4.
	buffer.SeekByte(((4 - offset%4) % 4), true)

	file.Audio.Header.BGMTrackSize = buffer.ReadU32LENext(1)[0]
	file.Audio.Header.SE1TrackSize = buffer.ReadU32LENext(1)[0]
	file.Audio.Header.SE2TrackSize = buffer.ReadU32LENext(1)[0]
	file.Audio.Header.SE3TrackSize = buffer.ReadU32LENext(1)[0]
	currentFrameSpeed := buffer.ReadByteNext()

	file.Audio.Header.CurrentFrameSpeed = byte(8) - currentFrameSpeed

	recordingBGMFrameSpeed := buffer.ReadByteNext()

	file.Audio.Header.RecordingBGMFrameSpeed = byte(8) - recordingBGMFrameSpeed

	if val, ok := ppmFramerates[file.Audio.Header.CurrentFrameSpeed]; ok {
		file.Framerate = val
	}

	if val, ok := ppmFramerates[file.Audio.Header.RecordingBGMFrameSpeed]; ok {
		file.BGMRate = val
	}

	buffer.SeekByte(14, true)

	file.Audio.Data.RawBGM = buffer.ReadBytesNext(int64(file.Audio.Header.BGMTrackSize))
	file.Audio.Data.RawSE1 = buffer.ReadBytesNext(int64(file.Audio.Header.SE1TrackSize))
	file.Audio.Data.RawSE2 = buffer.ReadBytesNext(int64(file.Audio.Header.SE2TrackSize))
	file.Audio.Data.RawSE3 = buffer.ReadBytesNext(int64(file.Audio.Header.SE3TrackSize))

	if buffer.ByteOffset() == buffer.ByteCapacity() {
		return file, fmt.Errorf("this ppm file is unsigned. will not play on a dsi")
	}
	file.Signature = buffer.ReadBytesNext(0x80)

	//padding
	buffer.SeekByte(0x10, true)

	if buffer.ByteOffset() < buffer.ByteCapacity() {
		return nil, errors.New(fmt.Sprintf("unexpected data (%d) after signature", buffer.ByteCapacity()-buffer.ByteOffset()))
	}

	for {
		if file.FramesParsed >= file.FrameCount {
			break
		}
	}

	return file, nil
}

func (file *PPMFile) ParseFrame(frame uint16, data []byte) {
	file.Frames[frame] = ReadFrame(crunch.NewBuffer(data))
	file.FramesParsed++
}

func (file *PPMFile) ParseFrames(data []byte, offsets []uint32) {
	perGor := uint16(100)
	for i := uint16(0); i < uint16(len(offsets)); i += perGor {
		go func(i uint16) {
			for j := uint16(0); j < perGor; j++ {
				frame := i + j
				if frame >= uint16(len(offsets)) {
					break
				}

				frameOffset := 0x06A0 + 8 + int64(file.FrameOffsetTableSize) + int64(offsets[frame])
				frameEndset := frameOffset + (int64(file.AnimationDataSize) - int64(offsets[frame]))
				file.ParseFrame(frame, data[frameOffset:frameEndset])
			}
		}(i)
		time.Sleep(time.Millisecond * 3)
	}

	for {
		if file.FramesParsed >= uint16(len(offsets)) {
			break
		}
	}

	for i := uint16(1); i < uint16(len(offsets)); i++ {
		file.Frames[i].Overwrite(file.Frames[i-1])
	}
}

func CreateFile(author *Author, frames []*Frame, audio []byte) (*PPMFile, error) {
	file := &PPMFile{}
	file.FrameCount = uint16(len(frames) - 1)
	file.FormatVersion = 0x24

	file.RootAuthor = author
	file.ParentAuthor = author
	file.CurrentAuthor = author

	authorIdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(authorIdBytes, author.Id)

	mac6 := fmt.Sprintf("%x", utils.Reverse(utils.Take(authorIdBytes, 3)))
	//os := runtime.GOOS
	dt := time.Now()
	fnVM := fmt.Sprintf("%d", runtime.NumCPU())
	fnVm := string(runtime.Version())
	fnYY := dt.Year() - 2009
	fnMd := dt.Year()*32 + dt.Day()
	fnTi := (((dt.Hour()*3600 + dt.Minute()*60 + dt.Second()) % 4096) >> 1)
	if fnMd > 255 {
		fnTi += 1
	}
	fnYMD := (fnYY << 9) + fnMd
	H6_9 := fmt.Sprintf("%x", fnYMD)
	//H89 := fmt.Sprintf("%x", fnMd)
	HABC := fmt.Sprintf("%x", fnTi)

	str13 := fmt.Sprintf("80%s%s%s%s", fnVM, fnVm, H6_9, HABC)
	//nEdited := fmt.Sprintf("%03d", 0)
	//fileName := fmt.Sprintf("%s_%s_%s.ppm", mac6, str13, nEdited)

	rawfn := make([]byte, 18)
	for i := 0; i < 3; i++ {
		a, err := strconv.Atoi(fmt.Sprintf("%d", mac6[2*i]+mac6[2*i+1]))
		if err != nil {
			return nil, err
		}

		rawfn[i] = byte(a)
	}

	for i := 3; i < 16; i++ {
		rawfn[i] = byte(str13[i-3])
	}

	rawfn[16] = 0
	rawfn[17] = 0

	var err error

	file.ParentFilename, err = FilenameFrom(rawfn)
	if err != nil {
		return nil, err
	}
	file.CurrentFilename, err = FilenameFrom(rawfn)
	if err != nil {
		return nil, err
	}

	var byteRootFileFragment = make([]byte, 8)
	for i := 0; i < 3; i++ {
		conv, err := strconv.Atoi(fmt.Sprintf("%d", mac6[2*i]+mac6[2*i+1]))
		if err != nil {
			return nil, err
		}
		byteRootFileFragment[i] = byte(conv)
	}

	for i := 3; i < 8; i++ {
		conv1, err := strconv.Atoi(fmt.Sprintf("%d", str13[2*(i-3)]))
		if err != nil {
			return nil, err
		}

		conv1 = conv1 << 4

		conv2, err := strconv.Atoi(fmt.Sprintf("%d", str13[2*(i-3)+1]))
		if err != nil {
			return nil, err
		}

		byteRootFileFragment[i] = byte(conv1 + conv2)
	}

	copy(file.RootFileFragment[:], byteRootFileFragment)
	jan, err := time.Parse("2006-02-01", "2000-01-01")
	if err != nil {
		return nil, err
	}
	file.Timestamp = NewTimestamp(uint32(time.Since(jan).Seconds()))
	file.ThumbnailFrameIndex = 0

	animationDataSize := uint32(8 + 4*len(frames))
	file.AnimationFlags = 0x43
	file.FrameOffsetTableSize = uint16(len(frames) * 4)

	file.Frames = make([]*Frame, len(frames))
	for i, frame := range frames {
		file.Frames[i] = frame

		animationDataSize += uint32(len(frame.Bytes()))
	}

	for (animationDataSize & 0x3) != 0 {
		animationDataSize++
	}

	file.AnimationDataSize = animationDataSize
	file.Audio = NewAudio()
	file.Audio.Data.RawBGM = audio
	file.Audio.Header.BGMTrackSize = uint32(len(audio))
	file.Audio.Header.SE1TrackSize = 0
	file.Audio.Header.SE2TrackSize = 0
	file.Audio.Header.SE3TrackSize = 0

	file.Audio.Header.CurrentFrameSpeed = 0
	file.Audio.Header.RecordingBGMFrameSpeed = 0

	file.SoundDataSize = uint32(len(audio))
	file.Thumbnail = make([]byte, 1536)

	return file, nil
}

var magic = []uint8{'P', 'A', 'R', 'A'}

func (f *PPMFile) Save(path string) error {

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	binary.Write(file, binary.LittleEndian, magic)
	binary.Write(file, binary.LittleEndian, f.AnimationDataSize)
	binary.Write(file, binary.LittleEndian, f.SoundDataSize)
	binary.Write(file, binary.LittleEndian, f.FrameCount)
	binary.Write(file, binary.LittleEndian, uint16(0x24))
	if f.Locked {
		binary.Write(file, binary.LittleEndian, uint16(1))
	} else {
		binary.Write(file, binary.LittleEndian, uint16(0))
	}
	binary.Write(file, binary.LittleEndian, f.ThumbnailFrameIndex)

	goRootAuthor := []byte(f.RootAuthor.Name)
	//fmt.Println(string(goRootAuthor))

	goParentAuthor := []byte(f.ParentAuthor.Name)

	goCurrentAuthor := []byte(f.CurrentAuthor.Name)

	rootAuthor := utils.WriteUTF16String(string(goRootAuthor))
	parentAuthor := utils.WriteUTF16String(string(goParentAuthor))
	currentAuthor := utils.WriteUTF16String(string(goCurrentAuthor))

	for len(rootAuthor) != 11 {
		rootAuthor = append(rootAuthor, '\000')
	}
	for len(parentAuthor) != 11 {
		parentAuthor = append(parentAuthor, '\000')
	}
	for len(currentAuthor) != 11 {
		currentAuthor = append(currentAuthor, '\000')
	}
	binary.Write(file, binary.LittleEndian, rootAuthor)
	binary.Write(file, binary.LittleEndian, parentAuthor)
	binary.Write(file, binary.LittleEndian, currentAuthor)

	binary.Write(file, binary.LittleEndian, f.ParentAuthor.Id)
	binary.Write(file, binary.LittleEndian, f.CurrentAuthor.Id)
	binary.Write(file, binary.LittleEndian, f.ParentFilename.buffer)
	binary.Write(file, binary.LittleEndian, f.CurrentFilename.buffer)
	binary.Write(file, binary.LittleEndian, f.RootAuthor.Id)
	binary.Write(file, binary.LittleEndian, f.RootFileFragment)
	binary.Write(file, binary.LittleEndian, f.Timestamp.Value)
	binary.Write(file, binary.LittleEndian, uint16(0))
	binary.Write(file, binary.LittleEndian, f.Thumbnail)

	binary.Write(file, binary.LittleEndian, f.FrameOffsetTableSize)
	binary.Write(file, binary.LittleEndian, uint32(0))
	binary.Write(file, binary.LittleEndian, f.AnimationFlags)

	lst := make([][]byte, 0)
	offset := uint32(0)

	for i := 0; i < len(f.Frames); i++ {
		if i == 0 {
			f.Frames[i].FirstByteHeader |= 0x80
		}

		lst = append(lst, f.Frames[i].Bytes())
		binary.Write(file, binary.LittleEndian, offset)
		offset += uint32(len(lst[i]))
	}

	for i := 0; i < len(f.Frames); i++ {
		binary.Write(file, binary.LittleEndian, lst[i])
	}

	for pos, _ := file.Seek(0, io.SeekCurrent); (4-pos%4)%4 != 0; pos, _ = file.Seek(0, io.SeekCurrent) {
		binary.Write(file, binary.LittleEndian, byte(0x00))
	}

	for i := 0; i < len(f.Frames); i++ {
		binary.Write(file, binary.LittleEndian, byte(0))
	}

	for pos, _ := file.Seek(0, io.SeekCurrent); (4-pos%4)%4 != 0; pos, _ = file.Seek(0, io.SeekCurrent) {
		binary.Write(file, binary.LittleEndian, byte(0x00))
	}

	binary.Write(file, binary.LittleEndian, uint32(len(f.Audio.Data.RawBGM)))
	binary.Write(file, binary.LittleEndian, uint32(len(f.Audio.Data.RawSE1))) //SE1
	binary.Write(file, binary.LittleEndian, uint32(len(f.Audio.Data.RawSE2))) //SE2
	binary.Write(file, binary.LittleEndian, uint32(len(f.Audio.Data.RawSE3))) //SE3

	binary.Write(file, binary.LittleEndian, f.Audio.Header.CurrentFrameSpeed)
	binary.Write(file, binary.LittleEndian, f.Audio.Header.RecordingBGMFrameSpeed)
	binary.Write(file, binary.LittleEndian, make([]byte, 14))

	binary.Write(file, binary.LittleEndian, f.Audio.Data.RawBGM)
	binary.Write(file, binary.LittleEndian, f.Audio.Data.RawSE1)
	binary.Write(file, binary.LittleEndian, f.Audio.Data.RawSE2)
	binary.Write(file, binary.LittleEndian, f.Audio.Data.RawSE3)

	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	stats, err := file.Stat()
	if err != nil {
		return err
	}

	streamSize := stats.Size()

	file.Seek(0, io.SeekStart)
	toHash := make([]byte, streamSize)
	file.Read(toHash)
	file.Seek(pos, io.SeekStart)

	f.setSignature(toHash)

	binary.Write(file, binary.LittleEndian, f.Signature)
	binary.Write(file, binary.LittleEndian, make([]byte, 0x10))

	return nil
}

var ppmFramerates = map[byte]float32{
	0: 30.0,
	1: 0.5,
	2: 1.0,
	3: 2.0,
	4: 4.0,
	5: 6.0,
	6: 12.0,
	7: 20.0,
	8: 30.0,
}

func (f *PPMFile) setSignature(data []byte) {
	if f.Key == nil {
		f.Signature = make([]byte, 0x80)
		return
	}

	hashed := sha1.Sum(data)
	data, err := rsa.SignPKCS1v15(rand.Reader, f.Key, crypto.SHA1, hashed[:])
	if err != nil {
		return
	}

	f.Signature = data
}
