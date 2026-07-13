package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestMessageQueryParsesContentFilters(t *testing.T) {
	values := url.Values{
		"topic":         {"orders"},
		"partition":     {"2"},
		"mode":          {"latest"},
		"limit":         {"25"},
		"scanLimit":     {"5000"},
		"keyFilter":     {"order-"},
		"keyOperator":   {"prefix"},
		"valueFilter":   {"SUCCESS"},
		"valueOperator": {"contains"},
		"jsonFilters":   {`[{"path":"data.user.id","operator":"eq","value":"10086"}]`},
	}
	request := httptest.NewRequest(http.MethodGet, "/?"+values.Encode(), nil)
	query, err := messageQuery(request, "")
	if err != nil {
		t.Fatal(err)
	}
	if query.Topic != "orders" || query.Partition != 2 || query.Limit != 25 || query.ScanLimit != 5000 || query.KeyOperator != "prefix" || len(query.JSONFilters) != 1 || query.JSONFilters[0].Path != "data.user.id" {
		t.Fatalf("query=%+v", query)
	}
}

func TestMessageQueryRejectsMalformedJSONFilters(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?topic=orders&jsonFilters=not-json", nil)
	if _, err := messageQuery(request, "live"); err == nil {
		t.Fatal("accepted malformed JSON filters")
	}
}
