package internal

import (
	"errors"
	_const "github.com/zikwall/grower/pkg/const"
)

type Publisher interface {
	Publish(topic _const.Topic) (func(message _const.Message), error)
}

func (g *Grower) Publish(topic _const.Topic) (func(message _const.Message), error) {
	_, ok := g.messagePool[topic]

	if !ok {
		return nil, errors.New("topic not found")
	}

	return func(message _const.Message) {
		g.messagePool[topic] <- message
	}, nil
}
