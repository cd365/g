package es

import (
	"errors"
)

// MyNumErr Custom error with error number.
type MyNumErr interface {
	error

	// Msg Set custom message.
	Msg(msg string) MyNumErr

	// GetNum Get custom number.
	GetNum() string

	// GetErr Get raw error.
	GetErr() error

	// GetMsg Get custom message.
	GetMsg() string
}

type myNumErr struct {
	num string // custom error number
	err error  // raw error value
	msg string // custom message
}

func (s *myNumErr) Error() string {
	if s.err != nil {
		return s.err.Error()
	}
	if s.msg != "" {
		return s.msg
	}
	return s.num
}

func (s *myNumErr) Msg(msg string) MyNumErr {
	s.msg = msg
	return s
}

func (s *myNumErr) GetNum() string {
	return s.num
}

func (s *myNumErr) GetErr() error {
	return s.err
}

func (s *myNumErr) GetMsg() string {
	return s.msg
}

func NumErr(num string, err error) MyNumErr {
	var tmp MyNumErr
	if ok := errors.As(err, &tmp); ok {
		return tmp
	}
	tmp = &myNumErr{
		num: num,
		err: err,
	}
	return tmp
}

const (
	X00000000 = "0x00000000" // no error
	X00000001 = "0x00000001" // unknown error
)
