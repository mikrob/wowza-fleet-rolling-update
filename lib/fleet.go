package lib

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/log"
	"github.com/coreos/fleet/pkg"
	"github.com/coreos/fleet/ssh"
)

var (
	sshUsername = "core"
	sshHost     = "coreosdev0001.botsunit.io"
	endPoint    = "http://192.168.1.1:49153"
)

// getClient initializes a client of fleet based on CLI flags
func getClient() (client.API, error) {
	clientDriver, err := getHTTPClient()

	return clientDriver, err
}

func getTimeout(seconds int) time.Duration {
	return time.Duration(seconds*1000) * time.Millisecond
}

func getHTTPClient() (client.API, error) {
	log.EnableDebug()
	tunnelFunc := net.Dial
	ep, err := url.Parse(endPoint)
	if err != nil {
		return nil, err
	}

	sshClient, err := ssh.NewSSHClient(sshUsername, sshHost, nil, true, getTimeout(30))
	if err != nil {
		return nil, fmt.Errorf("failed initializing SSH client: %v", err)
	}
	tunnelFunc = sshClient.Dial
	dialFunc := tunnelFunc

	trans := pkg.LoggingHTTPTransport{
		Transport: http.Transport{
			Dial:            dialFunc,
			TLSClientConfig: nil,
		},
	}

	hc := http.Client{
		Transport: &trans,
	}

	return client.NewHTTPClient(&hc, *ep)
}

// ListFleetMachines allow to list machines with fleet
func ListFleetMachines() {
	cAPI, err := getClient()
	if err != nil {
		fmt.Printf("Unable to initialize client: %v", err)
		os.Exit(1)
	}
	machines, err := cAPI.Machines()
	if err != nil {
		fmt.Println("error while retrieving machines")
		fmt.Println(err.Error())
	}
	for _, machine := range machines {
		fmt.Printf("Machine ID : %s, IP : %s Roles : %s", machine.ID, machine.PublicIP, machine.Metadata)
		fmt.Println()
	}
}
