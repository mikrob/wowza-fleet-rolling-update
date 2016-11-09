package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
	"wowza-rolling-update/digest"
	"wowza-rolling-update/lib"

	"github.com/hashicorp/consul/api"
)

var (
	serviceName         = flag.String("service", "", "Consul service name")
	datacenterName      = flag.String("dc", "", "Consul datacenter")
	tagOpts             = flag.String("tag", "", "Tag (key=value)")
	addTagActionOpts    = flag.Bool("add-tag", false, "Add tag")
	deleteTagActionOpts = flag.Bool("delete-tag", false, "Delete tag")
	listActionOpts      = flag.Bool("list", false, "List services")
	unit                = flag.String("unit", "", "Unit to start")
	update              = flag.String("update", "", "Image to update to")
	unitsDir            = flag.String("units-dir", ".", "Path to directory of fleet unit files")
	fleetSSHServer      = flag.String("fleet-ssh-server", "", "A server to SSH for Fleet API")
	fleetSSHUser        = flag.String("fleet-ssh-user", "core", "SSH username")
)

func main() {
	transport := digest.NewTransport("admin", "admin.123")
	flag.Parse()

	if *listActionOpts {
		client, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			panic(err)
		}

		queryOpts := &api.QueryOptions{
			Datacenter: *datacenterName,
		}

		catalogServices, _, err := client.Catalog().Service(*serviceName, "", queryOpts)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, s := range catalogServices {
			cs := lib.CatalogService{Dc: *datacenterName, Cs: s}
			currentConnections, _ := lib.GetMetrics(cs.GetURL(), transport)
			fmt.Printf("[%s] node:%s lan:%s wan:%s tags:%s current_connections:%d\n",
				s.ServiceName,
				s.Node,
				s.TaggedAddresses["lan"],
				s.TaggedAddresses["wan"],
				s.ServiceTags,
				currentConnections.CurrentConnections,
			)
		}
	} else if *update != "" && *serviceName != "" && *datacenterName != "" && *unitsDir != "" && *fleetSSHServer != "" {
		unitPath := fmt.Sprintf("%s/%s@.service", *unitsDir, *serviceName)
		if _, err := os.Stat(unitPath); os.IsNotExist(err) {
			log.Println(err)
			os.Exit(1)
		}
		client, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			panic(err)
		}

		queryOpts := &api.QueryOptions{
			Datacenter: *datacenterName,
		}

		// loop start
		// search for a service without this image name and tag it for future update (primarily includes already tagged service)
		// wait until this service has no connections
		// search unit linked to that service
		// destroy unit
		// start unit
		// loop until all image tag are equals to update tag

		updateTag := lib.Tag{Key: "update", Value: *update}
		alreadyUpdatedTag := lib.Tag{Key: "image", Value: *update}

		loopIndex := 1
		for {
			time.Sleep(3 * time.Second)
			catalogServices, _, err := client.Catalog().Service(*serviceName, "", queryOpts)
			if err != nil {
				fmt.Println(err)
				break
			}
			// search if we already have a service already waiting for an update
			service, err := lib.SearchServiceWithTag(catalogServices, updateTag)
			if err != nil {
				service, err = lib.SearchServiceWithoutTag(catalogServices, alreadyUpdatedTag)
				if err != nil {
					log.Println(err)
					break
				}
			}
			cs := lib.CatalogService{Dc: *datacenterName, Cs: &service}
			if !cs.HasTag(updateTag) {
				err = cs.ServiceAddTag(client, &service, updateTag)
				if err != nil {
					log.Println(err)
					break
				}
			}
			log.Println("Found service")

			for {
				currentConnections, errs := lib.GetMetrics(cs.GetURL(), transport)
				if errs != nil {
					log.Println("Unable to retrieve wowza metrics for service", cs.Cs.ServiceName, cs.Cs.ServiceAddress, cs.GetURL())
					continue
				}
				log.Println(currentConnections.CurrentConnections, "connections left in", cs.Cs.ServiceName, cs.Cs.ServiceAddress)
				if currentConnections.CurrentConnections == 0 {
					break
				}
				time.Sleep(3 * time.Second)
			}
			currentConnectionsRenew, err := lib.GetMetrics(cs.GetURL(), transport)
			if err != nil {
				log.Println("Unable to retrieve wowza metrics for service", cs.Cs.ServiceName, cs.Cs.ServiceAddress, cs.GetURL())
				continue
			}
			if currentConnectionsRenew.CurrentConnections == 0 {
				log.Println(cs.Cs.ServiceName, cs.Cs.Address, cs.Cs.Node, currentConnectionsRenew.CurrentConnections, "connections")
				// search fleet machine
				unitList, _ := lib.ListFleetUnits(*fleetSSHUser, *fleetSSHServer)

				cAPI, err := lib.GetClient(*fleetSSHUser, *fleetSSHServer)
				if err != nil {
					fmt.Printf("Unable to initialize client: %v", err)
					continue
				}
				machines, err := cAPI.Machines()
				if err != nil {
					log.Println("error while retrieving machines")
					log.Println(err.Error())
				}
				for _, machine := range machines {
					// select machine where service is running
					if machine.PublicIP == cs.Cs.Address {
						for _, unit := range unitList {
							unitFound, _ := regexp.MatchString(fmt.Sprintf("%s@.*.service", *serviceName), unit.Name)
							if unit.MachineID == machine.ID && unitFound {
								units := []string{unit.Name}
								lib.RunDestroyUnit(units, &cAPI)
								log.Println("Destroyed unit", unit.Name, "on server", machine.PublicIP)
								time.Sleep(3 * time.Second)
								unitFile := fmt.Sprintf("%s/%s", *unitsDir, unit.Name)
								units = []string{unitFile}
								lib.RunStartUnit(units, &cAPI)
								log.Println("Start unit", unit.Name, "with file")
								time.Sleep(30 * time.Second)
							}
						}
					}
				}
			}
			loopIndex += loopIndex
		}

	} else {
		flag.Usage()
	}
	os.Exit(0)
	// var tag lib.Tag
	// if *tagOpts != "" {
	// 	tag.DeconstructTag(*tagOpts)
	// }
	//
	// client, err := api.NewClient(api.DefaultConfig())
	// if err != nil {
	// 	panic(err)
	// }
	//
	// queryOpts := &api.QueryOptions{
	// 	Datacenter: *datacenterName,
	// }
	//
	// catalogServices, _, err := client.Catalog().Service(*serviceName, "", queryOpts)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	//
	// if *listActionOpts {
	// 	for _, s := range catalogServices {
	// 		cs := lib.CatalogService{Dc: *datacenterName, Cs: s}
	// 		if *tagOpts != "" && !cs.HasTag(tag) {
	// 			continue
	// 		}
	// 		currentConnections, _ := lib.GetMetrics(cs.GetURL(), transport)
	// 		fmt.Printf("[%s] node:%s lan:%s wan:%s tags:%s current_connections:%d\n",
	// 			s.ServiceName,
	// 			s.Node,
	// 			s.TaggedAddresses["lan"],
	// 			s.TaggedAddresses["wan"],
	// 			s.ServiceTags,
	// 			currentConnections.CurrentConnections,
	// 		)
	// 	}
	// 	os.Exit(0)
	// } else if *addTagActionOpts {
	// 	service, err := lib.SearchServiceWithoutTag(catalogServices, tag)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		os.Exit(0)
	// 	}
	// 	cs := lib.CatalogService{Dc: *datacenterName, Cs: &service}
	// 	err = cs.ServiceAddTag(client, &service, tag)
	// 	if err != nil {
	// 	}
	// } else if *deleteTagActionOpts {
	// 	for _, service := range catalogServices {
	// 		cs := lib.CatalogService{Dc: *datacenterName, Cs: service}
	// 		cs.ServiceDeleteTag(client, service, tag)
	// 	}
	// } else {
	// 	flag.Usage()
	// }
	// machineList, _ := lib.ListFleetMachines()
	// lib.PrintMachineList(machineList)
	//
	// unitList, _ := lib.ListFleetUnits()
	// lib.PrintUnitList(unitList)
	//
	// cAPI, err := lib.GetClient()
	// if err != nil {
	// 	fmt.Printf("Unable to initialize client: %v", err)
	// 	os.Exit(1)
	// }
	// units := []string{*unit}
	// //lib.RunStartUnit(units, &cAPI)
	// lib.RunDestroyUnit(units, &cAPI)
}
