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
	client := http.DefaultClient

	client.Transport = newMockRoundTripper()

	resp, err := client.Get("http://ifconfig.co/all.json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("GET http://ifconfig.co/all.json")
	fmt.Println(string(body))
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
	    "currentConnections" : 3,
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
    "currentConnections" : 3,
		"maxIncommingStreams" : 2,
		"wowzaFieldInvented" : "INVENTION"
		}`
	response.Body = ioutil.NopCloser(strings.NewReader(responseBody))
	return response, nil
}
