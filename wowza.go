package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
)

func main() {
	transport := digest.NewTransport("admin", "admin.123")
	flag.Parse()

	if *update != "" && *serviceName != "" && *datacenterName != "" {
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
			service, err := lib.SearchServiceWithoutTag(catalogServices, alreadyUpdatedTag)
			if err != nil {
				fmt.Println(err)
			}
			cs := lib.CatalogService{Dc: *datacenterName, Cs: &service}
			err = cs.ServiceAddTag(client, &service, updateTag)
			if err != nil {
				fmt.Println(err)
			}
			currentConnections, _ := lib.GetMetrics(cs.GetURL(), transport)
			if currentConnections.CurrentConnections == 0 {
				log.Println(cs.Cs.ServiceName, cs.Cs.Address, cs.Cs.Node, currentConnections.CurrentConnections, "connections")
				// search fleet machine
				machines, _ := lib.ListFleetMachines()
				for _, machine := range machines {
					if machine.PublicIP == cs.Cs.Address {
						lib.RunDestroyUnit(units, &cAPI)
						lib.RunStartUnit()
					}
				}
				// units, _ := lib.ListFleetUnits()
				// for _, unit := range units {
				// 	if unit.
				// }
			}
			loopIndex += loopIndex
			time.Sleep(1 * time.Second)
		}

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
	machineList, _ := lib.ListFleetMachines()
	lib.PrintMachineList(machineList)

	unitList, _ := lib.ListFleetUnits()
	lib.PrintUnitList(unitList)

	cAPI, err := lib.GetClient()
	if err != nil {
		fmt.Printf("Unable to initialize client: %v", err)
		os.Exit(1)
	}
	units := []string{*unit}
	//lib.RunStartUnit(units, &cAPI)
	lib.RunDestroyUnit(units, &cAPI)
}
