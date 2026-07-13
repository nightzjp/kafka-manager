package message

import "testing"

func TestMatchRecordFiltersKeyAndRawValue(t *testing.T) {
	record := Record{Key: "order-2026-001", Value: `payment accepted for customer-42`}
	for _, query := range []Query{
		{KeyFilter: "2026", KeyOperator: "contains"},
		{KeyFilter: "order-", KeyOperator: "prefix"},
		{KeyFilter: "order-2026-001", KeyOperator: "exact"},
		{ValueFilter: "customer-42", ValueOperator: "contains"},
		{ValueFilter: record.Value, ValueOperator: "exact"},
	} {
		matched, invalidJSON := matchRecord(record, query)
		if !matched || invalidJSON {
			t.Fatalf("query=%+v matched=%v invalidJSON=%v", query, matched, invalidJSON)
		}
	}
	matched, _ := matchRecord(record, Query{KeyFilter: "missing", KeyOperator: "contains"})
	if matched {
		t.Fatal("accepted a non-matching key")
	}
}

func TestMatchRecordUsesTypedNestedJSONConditions(t *testing.T) {
	record := Record{Value: `{"data":{"user":{"id":10086,"active":true},"tags":["vip","beta"]},"status":"SUCCESS"}`}
	query := Query{JSONFilters: []JSONFilter{
		{Path: "data.user.id", Operator: "gte", Value: "10000"},
		{Path: "data.user.active", Operator: "eq", Value: "true"},
		{Path: "data.tags.0", Operator: "eq", Value: "vip"},
		{Path: "status", Operator: "contains", Value: "CESS"},
	}}
	matched, invalidJSON := matchRecord(record, query)
	if !matched || invalidJSON {
		t.Fatalf("matched=%v invalidJSON=%v", matched, invalidJSON)
	}
	query.JSONFilters = append(query.JSONFilters, JSONFilter{Path: "data.user.id", Operator: "lt", Value: "10"})
	matched, _ = matchRecord(record, query)
	if matched {
		t.Fatal("AND conditions accepted a failing numeric comparison")
	}
}

func TestMatchRecordHandlesMissingPathsAndInvalidJSON(t *testing.T) {
	matched, invalidJSON := matchRecord(Record{Value: `not-json`}, Query{JSONFilters: []JSONFilter{{Path: "id", Operator: "exists"}}})
	if matched || !invalidJSON {
		t.Fatalf("invalid JSON matched=%v invalidJSON=%v", matched, invalidJSON)
	}
	matched, invalidJSON = matchRecord(Record{Value: `{"id":null}`}, Query{JSONFilters: []JSONFilter{{Path: "id", Operator: "exists"}}})
	if !matched || invalidJSON {
		t.Fatalf("existing null path matched=%v invalidJSON=%v", matched, invalidJSON)
	}
	matched, _ = matchRecord(Record{Value: `{}`}, Query{JSONFilters: []JSONFilter{{Path: "id", Operator: "neq", Value: "1"}}})
	if matched {
		t.Fatal("missing path must not satisfy neq")
	}
	matched, invalidJSON = matchRecord(Record{Value: `{"id":1} trailing`}, Query{JSONFilters: []JSONFilter{{Path: "id", Operator: "eq", Value: "1"}}})
	if matched || !invalidJSON {
		t.Fatalf("trailing content matched=%v invalidJSON=%v", matched, invalidJSON)
	}
}

func TestValidateQueryRejectsUnsafeFilters(t *testing.T) {
	valid := Query{Topic: "orders", Partition: -1, Mode: "latest", Limit: 100, ScanLimit: 5000}
	cases := []Query{
		withQuery(valid, func(q *Query) { q.KeyFilter, q.KeyOperator = "x", "regex" }),
		withQuery(valid, func(q *Query) { q.ValueFilter, q.ValueOperator = "x", "prefix" }),
		withQuery(valid, func(q *Query) { q.JSONFilters = []JSONFilter{{Path: "", Operator: "eq", Value: "x"}} }),
		withQuery(valid, func(q *Query) { q.JSONFilters = []JSONFilter{{Path: "id", Operator: "regex", Value: "x"}} }),
		withQuery(valid, func(q *Query) { q.JSONFilters = make([]JSONFilter, 6) }),
		withQuery(valid, func(q *Query) { q.ScanLimit = 50001 }),
	}
	for _, query := range cases {
		if _, err := validateQuery(query); err == nil {
			t.Fatalf("accepted unsafe query: %+v", query)
		}
	}
}

func TestRecordCollectorTracksScanAndInvalidJSONWithoutBufferingNonMatches(t *testing.T) {
	collector := newRecordCollector(Query{Limit: 2, ScanLimit: 4, JSONFilters: []JSONFilter{{Path: "status", Operator: "eq", Value: "ok"}}})
	for _, record := range []Record{
		{Value: `not-json`},
		{Value: `{"status":"no"}`},
		{Value: `{"status":"ok"}`, Offset: 3},
		{Value: `{"status":"ok"}`, Offset: 4},
		{Value: `{"status":"ok"}`, Offset: 5},
	} {
		if !collector.add(record) {
			break
		}
	}
	result := collector.result()
	if result.Scanned != 4 || result.SkippedInvalidJSON != 1 || len(result.Items) != 2 || !result.ResultLimited || !result.ScanLimited {
		t.Fatalf("result=%+v", result)
	}
	if result.Items[0].Offset != 3 || result.Items[1].Offset != 4 {
		t.Fatalf("items=%+v", result.Items)
	}
}

func TestLatestFilteredCollectorKeepsNewestMatchesInTheScanWindow(t *testing.T) {
	collector := newRecordCollector(Query{Mode: "latest", Limit: 2, ScanLimit: 5, KeyFilter: "match", KeyOperator: "contains"})
	for offset, key := range []string{"match-1", "no", "match-3", "match-4", "match-5"} {
		collector.add(Record{Offset: int64(offset + 1), Timestamp: int64((offset + 1) * 1000), Key: key})
	}
	result := collector.result()
	if result.Scanned != 5 || result.Matched != 4 || len(result.Items) != 2 || result.Items[0].Offset != 5 || result.Items[1].Offset != 4 || !result.ResultLimited {
		t.Fatalf("result=%+v", result)
	}
}

func withQuery(query Query, change func(*Query)) Query { change(&query); return query }
