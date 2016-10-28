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
	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/pkg"
	"github.com/coreos/fleet/schema"
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
func ListFleetMachines() ([]machine.MachineState, error) {
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
	return machines, err
}

// PrintMachineList allow to print a machine list
func PrintMachineList(list []machine.MachineState) {
	for _, machine := range list {
		fmt.Printf("Machine ID : %s, IP : %s Roles : %s", machine.ID, machine.PublicIP, machine.Metadata)
		fmt.Println()
	}
}

// ListFleetUnits allow to list deployed fleetunits
func ListFleetUnits() ([]*schema.Unit, error) {
	cAPI, err := getClient()
	if err != nil {
		fmt.Printf("Unable to initialize client: %v", err)
		os.Exit(1)
	}
	units, err := cAPI.Units()
	if err != nil {
		fmt.Println("error while retrieving units")
		fmt.Println(err.Error())
	}
	return units, err

}

// PrintUnitList allow to print a list of units for debug
func PrintUnitList(list []*schema.Unit) {
	for _, unit := range list {
		fmt.Printf("----------------------- %s@%s --------------------------\n", unit.Name, unit.MachineID)
		fmt.Printf("Current State : %s, Desired State : %s", unit.CurrentState, unit.DesiredState)
		fmt.Println()
		fmt.Println("Unit options : ")
		for _, option := range unit.Options {
			fmt.Printf("Option Name : %s, Section : %s, Value : %s", option.Name, option.Section, option.Value)
			fmt.Println()
		}
		fmt.Println("-------------------------------------------------------------------")
	}
}

// CreateAndStartUnit allow to create and start a fleet unit
func CreateAndStartUnit(unit *schema.Unit) {
	cAPI, err := getClient()
	if err != nil {
		fmt.Printf("Unable to initialize client: %v", err)
		os.Exit(1)
	}
	err = cAPI.CreateUnit(unit)
	if err != nil {
		fmt.Println("error while creating unit", unit.Name)
		fmt.Println(err.Error())
		os.Exit(0)
	}
}
