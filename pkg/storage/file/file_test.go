package file

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	t.Run("it should be successful create temp file", func(t *testing.T) {
		file, err := ioutil.TempFile("./", "prefix")

		if err != nil {
			log.Fatal(err)
		}

		t.Run("it should be successful write lines to file", func(t *testing.T) {
			for i := 1; i <= 10; i++ {
				if _, err := file.WriteString(fmt.Sprintf("Line %d\n", i)); err != nil {
					t.Fatal(err)
				}
			}
		})

		t.Run("it should be successful read chunks from file", func(t *testing.T) {
			lines, err := Read(file, 3, 7)

			if err != nil {
				t.Fatal(err)
			}

			if len(lines) != 5 {
				t.Fatal("Failed, expected five lines")
			}
		})

		_ = file.Close()

		if err = os.Remove(file.Name()); err != nil {
			t.Fatal(err)
		}
	})
}
