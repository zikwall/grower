package config

import "time"

type Buffer struct {
	BufSize          uint
	BufFlushInterval uint
}

type Runtime struct {
	Parallelism  int
	WriteTimeout time.Duration
	Debug        bool
}
