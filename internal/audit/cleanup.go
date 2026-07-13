package audit

import (
	"os"
	"path/filepath"
	"time"
)

func Cleanup(directory string, retentionDays int, now time.Time) error {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		date, err := time.ParseInLocation("2006-01-02", entry.Name(), now.Location())
		if err != nil {
			continue
		}
		if date.Before(cutoff) {
			if err := os.RemoveAll(filepath.Join(directory, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
