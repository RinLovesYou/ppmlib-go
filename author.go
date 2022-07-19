package ppmlib

import "errors"

type Author struct {
	Name string
	Id   uint64
}

func NewAuthor(name string, id uint64) (*Author, error) {
	if name == "" {
		return nil, errors.New("invalid author name")
	}

	if id == 0 {
		return nil, errors.New("invalid author id")
	}

	return &Author{
		Name: name,
		Id:   id,
	}, nil
}
