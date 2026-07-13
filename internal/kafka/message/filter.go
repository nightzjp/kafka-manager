package message

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strconv"
	"strings"
)

func hasContentFilters(query Query) bool {
	return query.KeyFilter != "" || query.ValueFilter != "" || len(query.JSONFilters) > 0
}

type recordCollector struct {
	query  Query
	output QueryResult
}

func newRecordCollector(query Query) *recordCollector {
	return &recordCollector{query: query, output: QueryResult{Items: make([]Record, 0, query.Limit)}}
}

func (collector *recordCollector) done() bool {
	if collector.output.Scanned >= collector.query.ScanLimit {
		return true
	}
	return !(collector.query.Mode == "latest" && hasContentFilters(collector.query)) && len(collector.output.Items) >= collector.query.Limit
}

func (collector *recordCollector) remainingScan() int {
	remaining := collector.query.ScanLimit - collector.output.Scanned
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (collector *recordCollector) add(record Record) bool {
	if collector.done() {
		return false
	}
	collector.output.Scanned++
	matched, invalidJSON := matchRecord(record, collector.query)
	if invalidJSON {
		collector.output.SkippedInvalidJSON++
	}
	if matched {
		collector.output.Matched++
		if collector.query.Mode == "latest" && hasContentFilters(collector.query) && len(collector.output.Items) >= collector.query.Limit {
			oldest := 0
			for index := 1; index < len(collector.output.Items); index++ {
				if recordNewer(collector.output.Items[oldest], collector.output.Items[index]) {
					oldest = index
				}
			}
			if recordNewer(record, collector.output.Items[oldest]) {
				collector.output.Items[oldest] = record
			}
		} else {
			collector.output.Items = append(collector.output.Items, record)
		}
	}
	return !collector.done()
}

func (collector *recordCollector) result() QueryResult {
	collector.output.ResultLimited = collector.output.Matched >= collector.query.Limit
	if collector.query.Mode == "latest" && hasContentFilters(collector.query) {
		collector.output.ResultLimited = collector.output.Matched > len(collector.output.Items)
		sort.SliceStable(collector.output.Items, func(i, j int) bool { return recordNewer(collector.output.Items[i], collector.output.Items[j]) })
	}
	collector.output.ScanLimited = collector.output.Scanned >= collector.query.ScanLimit
	return collector.output
}

func recordNewer(left, right Record) bool {
	if left.Timestamp != right.Timestamp {
		return left.Timestamp > right.Timestamp
	}
	if left.Partition != right.Partition {
		return left.Partition > right.Partition
	}
	return left.Offset > right.Offset
}

func matchRecord(record Record, query Query) (bool, bool) {
	if query.KeyFilter != "" && !matchText(record.Key, query.KeyFilter, query.KeyOperator) {
		return false, false
	}
	if query.ValueFilter != "" && !matchText(record.Value, query.ValueFilter, query.ValueOperator) {
		return false, false
	}
	if len(query.JSONFilters) == 0 {
		return true, false
	}
	decoder := json.NewDecoder(bytes.NewBufferString(record.Value))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return false, true
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return false, true
	}
	for _, filter := range query.JSONFilters {
		actual, exists := jsonPath(value, filter.Path)
		if !matchJSON(actual, exists, filter) {
			return false, false
		}
	}
	return true, false
}

func matchText(actual, expected, operator string) bool {
	switch operator {
	case "exact":
		return actual == expected
	case "prefix":
		return strings.HasPrefix(actual, expected)
	default:
		return strings.Contains(actual, expected)
	}
}

func jsonPath(value any, path string) (any, bool) {
	current := value
	for _, segment := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = typed[segment]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func matchJSON(actual any, exists bool, filter JSONFilter) bool {
	if filter.Operator == "exists" {
		return exists
	}
	if !exists {
		return false
	}
	switch filter.Operator {
	case "eq":
		return equalJSON(actual, filter.Value)
	case "neq":
		return !equalJSON(actual, filter.Value)
	case "contains":
		return strings.Contains(jsonText(actual), filter.Value)
	case "gt", "gte", "lt", "lte":
		actualNumber, actualOK := numberValue(actual)
		expectedNumber, err := strconv.ParseFloat(filter.Value, 64)
		if !actualOK || err != nil {
			return false
		}
		switch filter.Operator {
		case "gt":
			return actualNumber > expectedNumber
		case "gte":
			return actualNumber >= expectedNumber
		case "lt":
			return actualNumber < expectedNumber
		default:
			return actualNumber <= expectedNumber
		}
	}
	return false
}

func equalJSON(actual any, expected string) bool {
	switch typed := actual.(type) {
	case nil:
		return expected == "null"
	case string:
		return typed == expected
	case bool:
		parsed, err := strconv.ParseBool(expected)
		return err == nil && typed == parsed
	case json.Number:
		actualNumber, actualErr := typed.Float64()
		expectedNumber, expectedErr := strconv.ParseFloat(expected, 64)
		return actualErr == nil && expectedErr == nil && actualNumber == expectedNumber
	case float64:
		expectedNumber, err := strconv.ParseFloat(expected, 64)
		return err == nil && typed == expectedNumber
	default:
		return jsonText(actual) == expected
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		value, err := typed.Float64()
		return value, err == nil
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func jsonText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
