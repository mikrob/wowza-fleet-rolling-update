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
	endPoint = "unix:///var/run/fleet.sock"
)

// GetClient initializes a client of fleet based on CLI flags
func GetClient(sshUsername string, sshHost string) (client.API, error) {
	clientDriver, err := getHTTPClient(sshUsername, sshHost)

	return clientDriver, err
}

func getTimeout(seconds int) time.Duration {
	return time.Duration(seconds*1000) * time.Millisecond
}

func getHTTPClient(sshUsername string, sshHost string) (client.API, error) {
	log.EnableDebug()
	tunnelFunc := net.Dial
	ep, err := url.Parse(endPoint)
	if err != nil {
		return nil, err
	}
	dialUnix := ep.Scheme == "unix" || ep.Scheme == "file"
	sshClient, err := ssh.NewSSHClient(sshUsername, sshHost, nil, true, getTimeout(30))
	if err != nil {
		return nil, fmt.Errorf("failed initializing SSH client: %v", err)
	}
	if dialUnix {
		tgt := ep.Path
		tunnelFunc = func(string, string) (net.Conn, error) {
			log.Debugf("Establishing remote fleetctl proxy to %s", tgt)
			cmd := fmt.Sprintf(`fleetctl fd-forward %s`, tgt)
			return ssh.DialCommand(sshClient, cmd)
		}
	} else {
		tunnelFunc = sshClient.Dial
	}
	dialFunc := tunnelFunc

	if dialUnix {
		// This commonly happens if the user misses the leading slash after the scheme.
		// For example, "unix://var/run/fleet.sock" would be parsed as host "var".
		if len(ep.Host) > 0 {
			return nil, fmt.Errorf("unable to connect to host %q with scheme %q", ep.Host, ep.Scheme)
		}

		// The Path field is only used for dialing and should not be used when
		// building any further HTTP requests.
		ep.Path = ""

		// If not tunneling to the unix socket, http.Client will dial it directly.
		// http.Client does not natively support dialing a unix domain socket, so the
		// dial function must be overridden.

		// http.Client doesn't support the schemes "unix" or "file", but it
		// is safe to use "http" as dialFunc ignores it anyway.
		ep.Scheme = "http"

		// The Host field is not used for dialing, but will be exposed in debug logs.
		ep.Host = "domain-sock"
	}

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
func ListFleetMachines(sshUsername string, sshHost string) ([]machine.MachineState, error) {
	cAPI, err := GetClient(sshUsername, sshHost)
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
func ListFleetUnits(sshUsername string, sshHost string) ([]*schema.Unit, error) {
	cAPI, err := GetClient(sshUsername, sshHost)
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
func CreateAndStartUnit(unit *schema.Unit, sshUsername string, sshHost string) {
	cAPI, err := GetClient(sshUsername, sshHost)
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
