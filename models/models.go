package models

import "errors"

var ErrDBModelShouldNotBePointer = errors.New("model should not be a pointer")

type Models interface {
	Flag
}
