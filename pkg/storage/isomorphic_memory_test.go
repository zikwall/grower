package storage

import (
	"context"
	"testing"
	"time"
)

func TestNewIsomorphicMemoryStorage(t *testing.T) {
	t.Run("it should be create storage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		is := NewIsomorphicMemoryStorage(ctx)

		t.Run("it should be create first topic and write it", func(t *testing.T) {
			if err := is.NewTopic("rainbow", 2); err != nil {
				t.Fatal(err)
			}

			is.Write("rainbow", 2, "Its first message")
			is.Write("rainbow", 1, "Its second message")
			is.Write("rainbow", 2, "Its third message")
			is.Write("rainbow", 1, "Its four message")
			is.Write("rainbow", 2, "Its five message")

			<-time.After(1100 * time.Millisecond)

			is.mu.RLock()
			status := len(is.memory["rainbow"][1]) != 0 || len(is.memory["rainbow"][2]) != 0
			is.mu.RUnlock()

			if status {
				t.Fatal("Failed, expect empty partitions")
			}

			t.Run("it should be successful read from storage", func(t *testing.T) {
				messages, err := is.Read("rainbow", 2, 0, 3)

				if err != nil {
					t.Fatal(err)
				}

				if len(messages) != 3 {
					t.Fatalf("Failed, expect 2 items, give %d", len(messages))
				}

				messages, err = is.Read("rainbow", 1, 0, 1)

				if err != nil {
					t.Fatal(err)
				}

				if len(messages) != 1 {
					t.Fatalf("Failed, expect 2 items, give %d", len(messages))
				}
			})

			is.Write("rainbow", 2, "Another first message")
			is.Write("rainbow", 2, "Another second message")
			is.Write("rainbow", 1, "Another third message")
			is.Write("rainbow", 1, "Another four message")
			is.Write("rainbow", 1, "Another five message")

			_ = is.Close()

			is.mu.RLock()
			status = len(is.memory["rainbow"][1]) != 0 || len(is.memory["rainbow"][2]) != 0
			is.mu.RUnlock()

			if status {
				t.Fatal("Failed, expect empty partitions")
			}
		})
	})
}
