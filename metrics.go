package main

import (
	"encoding/json"
	"fmt"
	"wowza-rolling-update/digest"
)

// WowzaMetrics struct maps to the JSON automatically with the added meta data
// We only map the needed fields
type WowzaMetrics struct {
	MaxConnections      int32 `json:"maxConnections"`
	CurrentConnections  int32 `json:"currentConnections"`
	MaxIncommingStreams int32 `json:"maxIncommingStreams"`
}

var (
	url = "http://coreosdev0001.botsunit.io:8087/v2/servers/_defaultServer_/status"
)

func getMetrics(url string, transport *digest.Transport) (WowzaMetrics, error) {

	// initialize the client
	client, err := transport.Client()
	if err != nil {
		fmt.Println(err.Error())
	}

	// make the call (auth will happen)
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer resp.Body.Close()

	// used only for debug, warning it will clear the resp.Body buffer
	// body, err := ioutil.ReadAll(resp.Body)
	// fmt.Printf("Body: %v\n", string(body))

	var wowzaMetrics WowzaMetrics
	err = json.NewDecoder(resp.Body).Decode(&wowzaMetrics)
	if err != nil {
		fmt.Println("Cannot parse JSON from WOWZA because : ")
		fmt.Println(err.Error())
	}
	return wowzaMetrics, err
}

func fakeMain() {
	// setup a transport to handle disgest
	transport := digest.NewTransport("admin", "admin.123")
	wowzaMetrics, err := getMetrics(url, transport)
	if err != nil {
		fmt.Println("Failed to retrieve metrics with err : ", err.Error())
	}
	fmt.Println("Current Connections : ", wowzaMetrics.CurrentConnections)
}
