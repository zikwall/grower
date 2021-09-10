package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"sync"
)

type GrowerState struct {
	mu        sync.RWMutex
	consumers map[_const.Topic]map[_const.Group]map[_const.ConsumerUUID][]int
}

func NewGrowerState() *GrowerState {
	gs := &GrowerState{
		consumers: map[_const.Topic]map[_const.Group]map[_const.ConsumerUUID][]int{},
	}
	return gs
}
