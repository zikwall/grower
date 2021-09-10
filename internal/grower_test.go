package internal

import (
	"context"
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage"
	"testing"
	"time"
)

func TestNewGrower(t *testing.T) {
	t.Run("it should be create new topic with listeners", func(t *testing.T) {
		grower := NewGrower(context.Background(), &MockStorage{})

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

	t.Run("it should be successful test subscriber and publisher", func(t *testing.T) {
		grower := NewGrower(context.Background(), storage.NewIsomorphicMemoryStorage(context.Background()))

		if err := grower.CreateTopic("rainbow", 2); err != nil {
			t.Fatal(err)
		}

		publish, err := grower.Publish("rainbow")

		if err != nil {
			t.Fatal(err)
		}

		var savedMessages []string

		unsubscribe := grower.Subscribe("rainbow", "SOAP", func(messages ..._const.Message) {
			savedMessages = append(savedMessages, messages...)
		})

		publish("first")
		publish("second")
		publish("third")
		publish("four")
		publish("five")
		publish("six")

		<-time.After(time.Millisecond * 650)

		if len(savedMessages) != 6 {
			t.Fatalf("Failed, except 6 messages, give %d", len(savedMessages))
		}

		publish("seven")

		<-time.After(time.Millisecond * 650)

		if len(savedMessages) != 7 {
			t.Fatalf("Failed, except 7 messages, give %d", len(savedMessages))
		}

		publish("8")
		publish("9")

		<-time.After(time.Millisecond * 650)

		if len(savedMessages) != 9 {
			fmt.Println(savedMessages)
			t.Fatalf("Failed, except 9 messages, give %d", len(savedMessages))
		}

		unsubscribe()

		if err := grower.Drop(); err != nil {
			t.Fatal(err)
		}
	})
}
