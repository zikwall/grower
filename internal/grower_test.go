package internal

import (
	"context"
	"testing"
	"time"
)

func TestNewGrower(t *testing.T) {
	grower := NewGrower(context.Background(), &MockStorage{})

	t.Run("it should be create new topic with listeners", func(t *testing.T) {
		if err := grower.CreateTopic("rainbow", 2); err != nil {
			t.Fatal(err)
		}

		go func() {
			for i := 0; i < 10; i++ {
				grower.Write("rainbow", "_const.Message")
			}
		}()

		<-time.After(1 * time.Second)

		if err := grower.Drop(); err != nil {
			t.Fatal(err)
		}
	})
}
