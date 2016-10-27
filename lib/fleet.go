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

// Above is  tentative to use registryClient but Failed
//etcd "github.com/coreos/etcd/client"

// func getRegistryClient() (client.API, error) {
// 	var dial func(string, string) (net.Conn, error)
// 	sshClient, sshErr := ssh.NewSSHClient(sshUsername, sshHost, nil, false, getTimeout(30))
// 	if sshErr != nil {
// 		return nil, fmt.Errorf("failed initializing SSH client: %v", sshErr)
// 	}
// 	dial = func(network, addr string) (net.Conn, error) {
// 		tcpaddr, err := net.ResolveTCPAddr(network, addr)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return sshClient.DialTCP(network, nil, tcpaddr)
// 	}
//
// 	trans := &http.Transport{
// 		Dial:            dial,
// 		TLSClientConfig: nil,
// 	}
// 	defaultEndpoint := "unix:///var/run/fleet.sock"
// 	endPoint := defaultEndpoint
// 	eCfg := etcd.Config{
// 		Endpoints:               strings.Split(endPoint, ","),
// 		Transport:               trans,
// 		HeaderTimeoutPerRequest: getTimeout(30),
// 	}
//
// 	eClient, err := etcd.New(eCfg)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	etcdKeyPrefix := registry.DefaultKeyPrefix
// 	etcdAPI := etcd.NewKeysAPI(eClient)
// 	reg := registry.NewEtcdRegistry(etcdAPI, etcdKeyPrefix)
//
// 	if msg, ok := checkVersion(reg); !ok {
// 		fmt.Println(msg)
// 	}
//
// 	return &client.RegistryClient{Registry: reg}, nil
// }

// // checkVersion makes a best-effort attempt to verify that fleetctl is at least as new as the
// // latest fleet version found registered in the cluster. If any errors are encountered or fleetctl
// // is >= the latest version found, it returns true. If it is < the latest found version, it returns
// // false and a scary warning to the user.
// func checkVersion(cReg registry.ClusterRegistry) (string, bool) {
// 	fv := version.SemVersion
// 	lv, err := cReg.LatestDaemonVersion()
// 	oldVersionWarning := `####################################################################
// WARNING: fleetctl (%s) is older than the latest registered
// version of fleet found in the cluster (%s). You are strongly
// recommended to upgrade fleetctl to prevent incompatibility issues.
// ####################################################################
// `
// 	if err != nil {
// 		log.Errorf("error attempting to check latest fleet version in Registry: %v", err)
// 	} else if lv != nil && fv.LessThan(*lv) {
// 		return fmt.Sprintf(oldVersionWarning, fv.String(), lv.String()), false
// 	}
// 	return "", true
// }
