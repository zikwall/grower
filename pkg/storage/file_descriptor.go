package storage

import (
	"bufio"
	"errors"
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage/file"
	"os"
	"path"
)

// for tests
const sectionSize = 3

type WriterCallback = func(topic _const.Topic, partition _const.Partition, section uint) (*File, error)

func buildDefaultWriterCallback(commitDir string) WriterCallback {
	return func(topic _const.Topic, partition _const.Partition, section uint) (*File, error) {
		dir, err := os.Stat(commitDir)
		if err != nil {
			return nil, err
		}

		if !dir.IsDir() {
			return nil, errors.New("oops... commit directory is not directory")
		}

		topicDir := path.Join(commitDir, topic)

		if _, err := os.Stat(topicDir); os.IsNotExist(err) {
			if err := os.Mkdir(topicDir, 0755); err != nil {
				return nil, err
			}
		}

		filepath := path.Join(topicDir, fmt.Sprintf("%d-%d.growerlog", partition, section))
		f, _ := os.Create(filepath)
		return &File{file: f}, err
	}
}

type File struct {
	file *os.File
}

func (f *File) Read(from, to int64) ([]string, error) {
	return file.Read(f.file, from, to)
}

type FileDescriptor struct {
	topic                string
	partition            int
	file                 *File
	currentSectionSize   uint
	createWriterCallback WriterCallback
}

func NewFileDescriptor(commitDir string, topic _const.Topic, partition _const.Partition) Descriptor {
	descriptor := &FileDescriptor{topic: topic, partition: partition}
	descriptor.createWriterCallback = buildDefaultWriterCallback(commitDir)
	descriptor.file, _ = descriptor.createWriterCallback(topic, partition, 0)

	return descriptor
}

func (fd *FileDescriptor) Write(messages []_const.Message) {
	writer := bufio.NewWriter(fd.File())

	for _, message := range messages {
		// check partition size
		if fd.currentSectionSize > sectionSize {
			// create new section
			fd.file, _ = fd.createWriterCallback(fd.topic, fd.partition, fd.currentSectionSize)
		}

		_, _ = writer.WriteString(message)
		// in future split message
		fd.currentSectionSize += 1
	}

	_, _ = writer.WriteString("\n")
	_ = writer.Flush()
}

func (fd *FileDescriptor) Read(from, to int64) ([]_const.Message, error) {
	return fd.file.Read(from, to)
}

func (fd *FileDescriptor) File() *os.File {
	return fd.file.file
}

func (fd *FileDescriptor) Close() error {
	return fd.file.file.Close()
}
