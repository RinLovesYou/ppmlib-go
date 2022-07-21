package utils

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func ReadUTF16String(data []byte) string {
	win16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	utf16bom := unicode.BOMOverride(win16le.NewDecoder())
	unicodeReader := transform.NewReader(bytes.NewReader(data), utf16bom)
	decoded, err := ioutil.ReadAll(unicodeReader)
	if err != nil {
		panic(err)
	}

	return string(decoded)
}
func WriteUTF16String(data string) []byte {
	win16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	utf16bom := unicode.BOMOverride(win16le.NewEncoder())
	unicodeWriter := transform.NewReader(bytes.NewReader([]byte(data)), utf16bom)
	encoded, err := ioutil.ReadAll(unicodeWriter)
	if err != nil {
		panic(err)
	}

	return encoded
}

func IndexOf[T comparable](array []T, value T) int {
	for i, v := range array {
		if v == value {
			return i
		}
	}

	return -1
}

type Number interface {
	int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 | float32 | float64 | int | uint | uintptr
}

func Clamp[T Number](n, l, h T) T {
	if n < l {
		return l
	}

	if n > h {
		return h
	}

	return n
}

type MyWriter struct {
	buf []byte
	pos int
}

func (m *MyWriter) Write(p []byte) (n int, err error) {
	minCap := m.pos + len(p)
	if minCap > cap(m.buf) { // Make sure buf has enough capacity:
		buf2 := make([]byte, len(m.buf), minCap+len(p)) // add some extra
		copy(buf2, m.buf)
		m.buf = buf2
	}
	if minCap > len(m.buf) {
		m.buf = m.buf[:minCap]
	}
	copy(m.buf[m.pos:], p)
	m.pos += len(p)
	return len(p), nil
}

func Take[K any](array []K, count int) []K {
	return array[:count]
}

func Reverse[K any](array []K) []K {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}

	return array
}

func (m *MyWriter) Seek(offset int64, whence int) (int64, error) {
	newPos, offs := 0, int(offset)
	switch whence {
	case io.SeekStart:
		newPos = offs
	case io.SeekCurrent:
		newPos = m.pos + offs
	case io.SeekEnd:
		newPos = len(m.buf) + offs
	}
	if newPos < 0 {
		return 0, errors.New("negative result pos")
	}
	m.pos = newPos
	return int64(newPos), nil
}

//generic function that splits an array into multiple subarrays. eg 2, where the amount of items is divided by 2
//4, where the amount of items is devided by 4, etc. returns an array of arrays.
func Split[T any](array []T, amountOfArrays int) [][]T {
	if amountOfArrays%2 != 0 {
		amountOfArrays++
	}

	var result [][]T
	originalLength := len(array)
	amountToTake := originalLength / amountOfArrays
	for i := 0; i < amountOfArrays; i++ {
		result = append(result, Take(array, amountToTake))
		array = array[amountToTake:]
	}

	return result
}
