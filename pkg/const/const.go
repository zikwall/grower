package _const

import (
	"errors"
)

var ErrorShutdownWithoutGracefulCompletion = errors.New("shutdown completed without graceful completion")

type Topic = string
type Partition = int
type Message = string
