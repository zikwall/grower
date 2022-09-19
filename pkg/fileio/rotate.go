package fileio

import (
	"fmt"
	"os"
	"time"

	"github.com/zikwall/grower/pkg/log"
)

type Rotator interface {
	Rotate() error
}

type RotateCallback func(file *os.File) error

type Rotate struct {
	file              string
	dir               string
	backupFiles       uint
	backupFilesMaxAge time.Duration
	callback          RotateCallback
	enableScratch     bool
	enableRotate      bool
	skipNginxReopen   bool
}

func (r *Rotate) Rotate() error {
	if r.enableScratch {
		// only for local development mode, each cyclo create new access.log from scratch
		// will be removed in future
		fromScratch(r.file, r.dir)
		log.Infof("%s will be generated from scratch and mounted in the directory: %s", r.file, r.dir)
	}
	defer func() {
		// if rotation option is enabled, we delete outdated log files
		if r.enableRotate {
			if err := clearBackupFiles(r.file, r.dir, r.backupFiles, r.backupFilesMaxAge); err != nil {
				log.Warning(err)
			}
		}
	}()
	newFile, err := capture(r.dir, r.file)
	if err != nil {
		return err
	}
	if !r.skipNginxReopen {
		if err := reopen(); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(newFile, os.O_RDONLY, 0o777)
	if err != nil {
		return fmt.Errorf("failed open log file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warning(err)
		}
		// if rotation is not enabled, just delete the current index file
		if !r.enableRotate {
			if err := os.Remove(newFile); err != nil {
				log.Warning(err)
			}
		}
	}()
	if err := r.callback(f); err != nil {
		return err
	}
	return nil
}

func New(
	file string,
	dir string,
	backupFiles uint,
	backupFilesMaxAge time.Duration,
	enableScratch bool,
	enableRotate bool,
	skipNginxReopen bool,
	callback RotateCallback,
) Rotator {
	return &Rotate{
		file:              file,
		dir:               dir,
		backupFiles:       backupFiles,
		backupFilesMaxAge: backupFilesMaxAge,
		callback:          callback,
		enableScratch:     enableScratch,
		enableRotate:      enableRotate,
		skipNginxReopen:   skipNginxReopen,
	}
}
