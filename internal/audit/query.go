package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Filter struct {
	From, To, ClusterID, Action, Result string
	Limit                               int
}

func Query(directory string, filter Filter) ([]Entry, error) {
	if filter.Limit < 1 {
		filter.Limit = 100
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	dirs, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []Entry
	for _, dir := range dirs {
		if !dir.IsDir() || filter.From != "" && dir.Name() < filter.From || filter.To != "" && dir.Name() > filter.To {
			continue
		}
		files, err := filepath.Glob(filepath.Join(directory, dir.Name(), "audit-*.jsonl"))
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			file, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			for scanner.Scan() {
				var entry Entry
				if json.Unmarshal(scanner.Bytes(), &entry) != nil {
					continue
				}
				if filter.ClusterID != "" && entry.ClusterID != filter.ClusterID || filter.Action != "" && !strings.Contains(entry.Action, filter.Action) || filter.Result != "" && entry.Result != filter.Result {
					continue
				}
				items = append(items, entry)
			}
			scanErr := scanner.Err()
			file.Close()
			if scanErr != nil {
				return nil, scanErr
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Timestamp.After(items[j].Timestamp) })
	if len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}
