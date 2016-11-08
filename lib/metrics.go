package lib

import (
	"encoding/json"
	"fmt"
	"wowza-rolling-update/digest"
)

// Metrics struct maps to the JSON automatically with the added meta data. We only map the needed fields
type Metrics struct {
	MaxConnections      int32 `json:"maxConnections"`
	CurrentConnections  int32 `json:"currentConnections"`
	MaxIncommingStreams int32 `json:"maxIncommingStreams"`
}

// GetMetrics allow to retrive wowza metrics with mock or really
func GetMetrics(url string, transport *digest.Transport) (Metrics, error) {

	var metrics Metrics
	// initialize the client
	client, err := transport.Client()
	if err != nil {
		return metrics, err
	}

	// make the call (auth will happen)
	resp, err := client.Get(url)
	if err != nil {
		return metrics, err
	}
	defer resp.Body.Close()

	// used only for debug, warning it will clear the resp.Body buffer
	// body, err := ioutil.ReadAll(resp.Body)
	// fmt.Printf("Body: %v\n", string(body))

	err = json.NewDecoder(resp.Body).Decode(&metrics)
	if err != nil {
		fmt.Println("Cannot parse JSON from WOWZA because : ")
		fmt.Println(err.Error())
	}
	return metrics, err
}

func fakeMain() {
	// setup a transport to handle disgest
	transport := digest.NewTransport("admin", "admin.123")
	coreOSUrl := "http://coreosdev0001.botsunit.io:8087/v2/servers/_defaultServer_/status"
	wowzaMetrics, err := GetMetrics(coreOSUrl, transport)
	if err != nil {
		fmt.Println("Failed to retrieve metrics with err : ", err.Error())
	}
	fmt.Println("Current Connections : ", wowzaMetrics.CurrentConnections)
}
