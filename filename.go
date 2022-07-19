package ppmlib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Filename struct {
	buffer []byte
}

func FilenameFrom(buf []byte) (*Filename, error) {
	if len(buf) != 18 {
		return nil, errors.New("invalid filename length")
	}

	return &Filename{buffer: buf}, nil
}

func NewFilename(name string) (*Filename, error) {
	if len(name) != 24 {
		return nil, errors.New("invalid filename length")
	}

	if ok, err := regexp.MatchString("[0-9,A-F]{6}_[0-9,A-F]{13}_\\d{3}", name); err != nil && !ok {
		return nil, errors.New("invalid filename format")
	}

	buffer := make([]byte, 18)
	for i := 0; i < 3; i++ {
		conv, err := strconv.Atoi(fmt.Sprintf("%x", name[2*i]+name[2*i+1]))
		if err != nil {
			return nil, err
		}
		buffer[i] = byte(conv)
	}

	for i, j := 3, 7; i < 16; i, j = 1+i, 1+j {
		buffer[i] = name[j]
	}

	//substring at 21
	b := binary.LittleEndian.Uint16([]byte(name[21:23]))
	buffer[16] = byte(b)
	b >>= 8
	buffer[17] = byte(b)

	return FilenameFrom(buffer)
}

func (f Filename) String() string {
	var result string
	for i := 0; i < 3; i++ {
		result += strings.ToUpper(fmt.Sprintf("%x", f.buffer[i]))
	}

	result += "_"

	for i := 3; i < 16; i++ {
		result += strings.ToUpper(string(f.buffer[i]))
	}

	result += "_"

	numEdits := f.buffer[17]<<4 | f.buffer[16]

	result += fmt.Sprintf("%03d", numEdits)

	return result
}
