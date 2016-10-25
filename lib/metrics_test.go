package lib

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"wowza-rolling-update/digest"
)

func TestRetrieveMetricsFillStruct(t *testing.T) {
	url := "http://this_is_a_fake_url.com/wowza"
	wowzaMetrics, err := GetMetrics(url, newMocktransport("admin", "toto"))
	if err != nil {
		panic(err)
	}

	if wowzaMetrics.CurrentConnections != 45 {
		t.Error("wowza metrics current connections is not 45")
	}

	if wowzaMetrics.MaxConnections != 12 {
		t.Error("wowza metrics current connections is not 42")
	}

	if wowzaMetrics.MaxIncommingStreams != 2 {
		t.Error("wowza metrics current connections is not 42")
	}

}

type mockTransport struct{}

func newMockRoundTripper() http.RoundTripper {
	return &mockTransport{}
}

func newMocktransport(username, password string) *digest.Transport {
	t := &digest.Transport{
		Username: username,
		Password: password,
	}
	t.Transport = newMockRoundTripper()
	return t
}

// Implement http.RoundTripper
func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create mocked http.Response
	response := &http.Response{
		Header:     make(http.Header),
		Request:    req,
		StatusCode: http.StatusOK,
	}
	response.Header.Set("Content-Type", "application/json")

	responseBody :=
		`{
		"version" : "12345678910",
    "maxConnections": 12,
    "currentConnections" : 45,
		"maxIncommingStreams" : 2,
		"wowzaFieldInvented" : "INVENTION"
		}`
	response.Body = ioutil.NopCloser(strings.NewReader(responseBody))
	return response, nil
}
