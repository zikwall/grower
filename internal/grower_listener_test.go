package internal

import (
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"testing"
	"time"
)

func TestNewListener(t *testing.T) {
	t.Run("it should be successful stopped", func(t *testing.T) {
		ch := make(chan _const.Message, 10)
		ln := NewListener(&MockStorage{}, ch, "rainbow", 1)

		go func() {
			for i := 1; i <= 10; i++ {
				ch <- fmt.Sprintf("message %d", i)
			}
		}()

		<-time.After(1 * time.Second)

		ln.stop()
	})
}
