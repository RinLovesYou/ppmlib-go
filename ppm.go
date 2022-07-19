package ppmlib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/RinLovesYou/ppmlib-go/utils"
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

	RootFileFragment [8]byte

	Timestamp *Timestamp

	Thumbnail [1536]byte `json:"-"`

	FrameOffsetTableSize uint16

	Frames []*Frame `json:"-"`

	AnimationFlags uint16

	Audio *Audio `json:"-"`

	Framerate        float32
	BGMRate          float32
	SoundEffectFlags []byte
	Signature        []byte
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
	buffer := bytes.NewReader(data)

	magic := make([]byte, 4)
	_, err := buffer.Read(magic)
	if err != nil {
		return nil, err
	}

	if string(magic) != "PARA" {
		return nil, errors.New("invalid file magic")
	}

	file := &PPMFile{}

	err = binary.Read(buffer, binary.LittleEndian, &file.AnimationDataSize)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.SoundDataSize)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.FrameCount)
	if err != nil {
		return nil, err
	}

	file.FrameCount++

	err = binary.Read(buffer, binary.LittleEndian, &file.FormatVersion)
	if err != nil {
		return nil, err
	}

	var isLocked uint16
	err = binary.Read(buffer, binary.LittleEndian, &isLocked)
	if err != nil {
		return nil, err
	}

	file.Locked = isLocked != 0

	err = binary.Read(buffer, binary.LittleEndian, &file.ThumbnailFrameIndex)
	if err != nil {
		return nil, err
	}

	rootName, err := utils.ReadWChars(buffer, 11)
	if err != nil {
		return nil, err
	}

	parentName, err := utils.ReadWChars(buffer, 11)
	if err != nil {
		return nil, err
	}

	currentName, err := utils.ReadWChars(buffer, 11)
	if err != nil {
		return nil, err
	}

	var parentId uint64
	var currentId uint64

	err = binary.Read(buffer, binary.LittleEndian, &parentId)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &currentId)
	if err != nil {
		return nil, err
	}

	parentNameBuffer := make([]byte, 18)
	currentNameBuffer := make([]byte, 18)

	_, err = buffer.Read(parentNameBuffer)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Read(currentNameBuffer)
	if err != nil {
		return nil, err
	}

	file.ParentFilename, err = FilenameFrom(parentNameBuffer)
	if err != nil {
		return nil, err
	}

	file.CurrentFilename, err = FilenameFrom(currentNameBuffer)
	if err != nil {
		return nil, err
	}

	var rootId uint64
	err = binary.Read(buffer, binary.LittleEndian, &rootId)
	if err != nil {
		return nil, err
	}

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

	fragment := make([]byte, 8)
	_, err = buffer.Read(fragment)
	if err != nil {
		return nil, err
	}

	copy(file.RootFileFragment[:], fragment)

	var timestamp uint32
	err = binary.Read(buffer, binary.LittleEndian, &timestamp)
	if err != nil {
		return nil, err
	}

	file.Timestamp = NewTimestamp(timestamp)

	var blank uint16
	err = binary.Read(buffer, binary.LittleEndian, &blank) //0x9E
	if err != nil {
		return nil, err
	}

	thumbnail := make([]byte, 1536)
	_, err = buffer.Read(thumbnail)
	if err != nil {
		return nil, err
	}

	copy(file.Thumbnail[:], thumbnail)

	err = binary.Read(buffer, binary.LittleEndian, &file.FrameOffsetTableSize)
	if err != nil {
		return nil, err
	}

	var blank2 uint32
	err = binary.Read(buffer, binary.LittleEndian, &blank2) //0x6A2 - always 0
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.AnimationFlags)
	if err != nil {
		return nil, err
	}

	oCnt := file.FrameOffsetTableSize/4 - 1

	animationOffset := make([]uint32, int(oCnt)+1)

	file.Frames = make([]*Frame, file.FrameCount)

	for i := 0; i <= int(oCnt); i++ {
		var offset uint32
		err = binary.Read(buffer, binary.LittleEndian, &offset)
		if err != nil {
			return nil, err
		}

		animationOffset[i] = offset
	}

	//calculate the current position of the buffer
	currentPosition, err := buffer.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	offset := currentPosition

	for i := 0; i <= int(oCnt); i++ {
		if offset+int64(animationOffset[i]) == 4288480943 {
			return nil, errors.New("data might be corrupted (possible memory pit?)")
		}

		buffer.Seek(offset+int64(animationOffset[i]), io.SeekStart)
		file.Frames[i], err = ReadFrame(buffer)
		if err != nil {
			return nil, err
		}

		file.Frames[i].AnimationIndex = utils.IndexOf(animationOffset, animationOffset[i])
		if i > 0 {
			file.Frames[i].Overwrite(file.Frames[i-1])
		}
	}

	if file.SoundDataSize == 0 {
		return file, nil
	}

	offset = 0x6A0 + int64(file.AnimationDataSize)
	buffer.Seek(offset, io.SeekStart)

	file.SoundEffectFlags = make([]byte, len(file.Frames))

	file.Audio = NewAudio()
	for i := 0; i < len(file.Frames); i++ {
		file.SoundEffectFlags[i], err = buffer.ReadByte()
		if err != nil {
			return nil, err
		}
	}
	offset += int64(len(file.Frames))

	align := make([]byte, ((4 - offset%4) % 4))
	//makes the next offset divisible by 4.
	err = binary.Read(buffer, binary.LittleEndian, align)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.Audio.Header.BGMTrackSize)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.Audio.Header.SE1TrackSize)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.Audio.Header.SE2TrackSize)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.LittleEndian, &file.Audio.Header.SE3TrackSize)
	if err != nil {
		return nil, err
	}

	currentFrameSpeed, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}

	file.Audio.Header.CurrentFrameSpeed = byte(8) - currentFrameSpeed

	recordingBGMFrameSpeed, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}

	file.Audio.Header.RecordingBGMFrameSpeed = byte(8) - recordingBGMFrameSpeed

	if val, ok := ppmFramerates[file.Audio.Header.CurrentFrameSpeed]; ok {
		file.Framerate = val
	}

	if val, ok := ppmFramerates[file.Audio.Header.RecordingBGMFrameSpeed]; ok {
		file.BGMRate = val
	}

	_, err = buffer.Read(make([]byte, 14))
	if err != nil {
		return nil, err
	}

	file.Audio.Data.RawBGM = make([]byte, file.Audio.Header.BGMTrackSize)
	file.Audio.Data.RawSE1 = make([]byte, file.Audio.Header.SE1TrackSize)
	file.Audio.Data.RawSE2 = make([]byte, file.Audio.Header.SE2TrackSize)
	file.Audio.Data.RawSE3 = make([]byte, file.Audio.Header.SE3TrackSize)

	_, err = buffer.Read(file.Audio.Data.RawBGM)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Read(file.Audio.Data.RawSE1)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Read(file.Audio.Data.RawSE2)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Read(file.Audio.Data.RawSE3)
	if err != nil {
		return nil, err
	}

	currentPosition, err = buffer.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	if currentPosition == buffer.Size() {
		return file, fmt.Errorf("this ppm file is unsigned. will not play on a dsi")
	}

	file.Signature = make([]byte, 0x80)
	_, err = buffer.Read(file.Signature)
	if err != nil {
		return nil, err
	}

	//padding
	_, err = buffer.Read(make([]byte, 0x10))
	if err != nil {
		return nil, err
	}

	_, err = buffer.ReadByte()
	if err == nil {
		return nil, errors.New("managed to read past expected end of file")
	}

	return file, nil
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
	file.Thumbnail = *new([1536]byte)

	return file, nil
}

var magic = []byte{'P', 'A', 'R', 'A'}

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
	binary.Write(file, binary.LittleEndian, uint16(0x0024))
	if f.Locked {
		binary.Write(file, binary.LittleEndian, uint16(1))
	} else {
		binary.Write(file, binary.LittleEndian, uint16(0))
	}
	binary.Write(file, binary.LittleEndian, f.ThumbnailFrameIndex)

	goRootAuthor := []byte(f.RootAuthor.Name)
	fmt.Println(string(goRootAuthor))

	goParentAuthor := []byte(f.ParentAuthor.Name)

	goCurrentAuthor := []byte(f.CurrentAuthor.Name)

	rootAuthor, _ := syscall.UTF16FromString(string(goRootAuthor))
	parentAuthor, _ := syscall.UTF16FromString(string(goParentAuthor))
	currentAuthor, _ := syscall.UTF16FromString(string(goCurrentAuthor))

	for len(rootAuthor) != 11 {
		rootAuthor = append(rootAuthor, 0)
	}
	for len(parentAuthor) != 11 {
		parentAuthor = append(parentAuthor, 0)
	}
	for len(currentAuthor) != 11 {
		currentAuthor = append(currentAuthor, 0)
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

	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	binary.Write(file, binary.LittleEndian, make([]byte, (4-pos%4)%4))

	for i := 0; i < len(f.Frames); i++ {
		binary.Write(file, binary.LittleEndian, byte(0))
	}

	pos, err = file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return err
	}

	binary.Write(file, binary.LittleEndian, make([]byte, (4-pos%4)%4))

	binary.Write(file, binary.LittleEndian, uint32(len(f.Audio.Data.RawBGM)))
	binary.Write(file, binary.LittleEndian, uint32(0)) //SE1
	binary.Write(file, binary.LittleEndian, uint32(0)) //SE2
	binary.Write(file, binary.LittleEndian, uint32(0)) //SE3

	binary.Write(file, binary.LittleEndian, f.Audio.Header.CurrentFrameSpeed)
	binary.Write(file, binary.LittleEndian, f.Audio.Header.RecordingBGMFrameSpeed)
	binary.Write(file, binary.LittleEndian, make([]byte, 14))

	binary.Write(file, binary.LittleEndian, f.Audio.Data.RawBGM)

	binary.Write(file, binary.LittleEndian, make([]byte, 0x80))
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
