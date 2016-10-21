package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"wowza-rolling-update/digest"
)

func TestRetrieveMetrics(t *testing.T) {
	url := "http://this_is_a_fake_url.com/wowza"
	wowzaMetrics, err := getMetrics(url, newMocktransport("admin", "toto"))
	if err != nil {
		panic(err)
	}
	t.Log("Current Connections : ")
	t.Log(wowzaMetrics.CurrentConnections)

}

// Create a HTTP server to return mocked response
func newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w,
			`{
			"version" : "12345678910",
	    "maxConnections": 12,
	    "currentConnections" : 42,
			"maxIncommingStreams" : 2,
			"wowzaFieldInvented" : "INVENTION"
			}`)
	}))
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
