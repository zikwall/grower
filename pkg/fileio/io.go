package fileio

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zikwall/grower/pkg/log"
)

const extension = ".growerlog"
const timeLayout = "2006_01_02_15_04_05"

func BuildGrowerLogName(original string) string {
	return fmt.Sprintf("%s-%s%s", original, time.Now().Format(timeLayout), extension)
}

func DeleteOutdatedBackupFiles(original, directory string, maxBackups uint, maxAge time.Duration) error {
	files, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed read nginx logs dir: %w", err)
	}
	backFiles := make([]backupFile, 0, maxBackups+5)
	// the original file name is used as a prefix
	original += "-"
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != extension {
			continue
		}
		// for example: access.log-2022_07_21_15_41_45.growerlog
		filename := file.Name()
		if !strings.HasPrefix(filename, original) {
			log.Warningf("mismatched prefix (%s) for %s", original, filename)
			continue
		}
		if !strings.HasSuffix(filename, extension) {
			log.Warningf("mismatched extension (%s) for %s", extension, filename)
			continue
		}
		timestamp := filename[len(original) : len(filename)-len(extension)]
		t, err := time.ParseInLocation(timeLayout, timestamp, time.Local)
		if err != nil {
			log.Warning(err)
			continue
		}
		backFiles = append(backFiles, backupFile{t, file})
	}
	sort.Sort(sortableFiles(backFiles))
	var backupIndex uint = 1
	var remove = make([]backupFile, 0, len(backFiles))
	for _, f := range backFiles {
		if maxBackups > 0 && backupIndex > maxBackups {
			remove = append(remove, f)
			continue
		}
		if time.Since(f.timestamp) > maxAge {
			remove = append(remove, f)
		}
		backupIndex++
	}
	for _, file := range remove {
		log.Infof("detects an outdated index file, will be deleted: %s", file.Name())
		if err := os.Remove(path.Join(directory, file.Name())); err != nil {
			log.Warning(err)
		}
	}
	return nil
}

type backupFile struct {
	timestamp time.Time
	os.DirEntry
}

// we sort the files in descending order so that last files can be trimmed to maximum value of possible backup files
type sortableFiles []backupFile

func (s sortableFiles) Less(i, j int) bool {
	return s[i].timestamp.After(s[j].timestamp)
}

func (s sortableFiles) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortableFiles) Len() int {
	return len(s)
}

// FromScratch it is intended only to simplify development mode
// FromScratch simply generates a new log file for each iteration
func FromScratch(source, directory string) {
	pwd, _ := os.Getwd()
	// nolint:gosec // it's not important here only for debug mode
	if err := exec.Command("cp", path.Join(pwd, "sample_test.log"), directory).Run(); err != nil {
		log.Warning(err)
	} else {
		_ = os.Rename(path.Join(directory, "sample_test.log"), path.Join(directory, source))
	}
}

// CheckFile we check if file exists and if we can manipulate it
func CheckFile(file string) error {
	if _, err := os.Stat(file); err != nil {
		if err == os.ErrNotExist {
			return fmt.Errorf("file doesn't exists: %w", err)
		}
		return err
	}
	return nil
}
