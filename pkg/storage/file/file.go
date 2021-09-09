package file

import (
	"bufio"
	"os"
)

func Read(file *os.File, from, to int) ([]string, error) {
	// ioutil.TempFile creates a temp file and opens the file for reading and writing
	// and returns the resulting *os.File (file descriptor).
	// So when you're writing inside the file, the pointer is moved to that offset, i.e.,
	// it's currently at the end of the file. But as your requirement is read from the file,
	// need to Seek back to the beginning or wherever desired offset using *os.File.Seek method.
	// So, adding tmpFile.Seek(0, 0) will give you the desired behaviour.
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	n := 0
	var buf []string

	for scanner.Scan() {
		n++

		if n < from {
			continue
		}

		if n > to {
			break
		}

		buf = append(buf, scanner.Text())
	}

	return buf, scanner.Err()
}
