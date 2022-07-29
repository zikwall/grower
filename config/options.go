package config

type Buffer struct {
	BufSize          uint
	BufFlushInterval uint
}

type Runtime struct {
	Parallelism int
	Debug       bool
}
