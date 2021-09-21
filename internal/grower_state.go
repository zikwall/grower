package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"sync"
)

type GrowerState struct {
	mu        sync.RWMutex
	consumers map[_const.Topic]map[_const.Group]map[_const.ConsumerUUID][]int
	offsets   map[_const.Topic]map[_const.Group]map[_const.Partition]int64
	waits     map[_const.Topic]map[_const.Group]*sync.WaitGroup
}

func NewGrowerState() *GrowerState {
	gs := &GrowerState{
		consumers: map[_const.Topic]map[_const.Group]map[_const.ConsumerUUID][]int{},
		offsets:   map[_const.Topic]map[_const.Group]map[_const.Partition]int64{},
		waits:     map[_const.Topic]map[_const.Group]*sync.WaitGroup{},
	}
	return gs
}
