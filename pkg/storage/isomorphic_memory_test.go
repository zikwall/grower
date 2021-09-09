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

			is.Write("rainbow", 1, "Its first message")
			is.Write("rainbow", 1, "Its second message")
			is.Write("rainbow", 2, "Its third message")
			is.Write("rainbow", 2, "Its four message")
			is.Write("rainbow", 1, "Its five message")

			<-time.After(1100 * time.Millisecond)

			if len(is.memory["rainbow"][1]) != 0 || len(is.memory["rainbow"][2]) != 0 {
				t.Fatal("Failed, expect empty partitions")
			}

			is.Write("rainbow", 2, "Another first message")
			is.Write("rainbow", 2, "Another second message")
			is.Write("rainbow", 1, "Another third message")
			is.Write("rainbow", 1, "Another four message")
			is.Write("rainbow", 1, "Another five message")

			_ = is.Close()

			if len(is.memory["rainbow"][1]) != 0 || len(is.memory["rainbow"][2]) != 0 {
				t.Fatal("Failed, expect empty partitions")
			}
		})
	})
}
