package ppmlib

import "time"

type Timestamp struct {
	Value uint32
}

func NewTimestamp(val uint32) *Timestamp {
	return &Timestamp{
		Value: val,
	}
}

func (t Timestamp) String() string {
	dummyTime, err := time.Parse("2006-02-01", "2000-01-01")
	if err != nil {
		return ""
	}

	dummyTime = dummyTime.Add(time.Duration(t.Value) * time.Second)
	return dummyTime.Format("01/02/2006 15:04:05")
}
