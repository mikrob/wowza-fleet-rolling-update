package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"wowza-rolling-update/digest"
	lib "wowza-rolling-update/lib"

	"github.com/hashicorp/consul/api"
)

var (
	serviceName         = flag.String("service", "", "Consul service name")
	datacenterName      = flag.String("dc", "", "Consul datacenter")
	tagOpts             = flag.String("tag", "", "Tag (key=value)")
	addTagActionOpts    = flag.Bool("add-tag", false, "Add tag")
	deleteTagActionOpts = flag.Bool("delete-tag", false, "Delete tag")
	listActionOpts      = flag.Bool("list", false, "List services")
)

// Tag for a service
type Tag struct {
	Key   string
	Value string
}

func split(s, sep string) (string, string) {
	x := strings.Split(s, sep)
	return x[0], x[1]
}

func (t Tag) buildTag() string {
	return fmt.Sprintf("%s=%s", t.Key, t.Value)
}

func (t *Tag) deconstructTag(tag string) {
	k, v := split(tag, "=")
	t.Key = k
	t.Value = v
}

// CatalogService extends api.CatalogService
type CatalogService struct {
	Cs *api.CatalogService
	Dc string
}

func (cs *CatalogService) getURL() string {
	url := fmt.Sprintf("http://%s.botsunit.io:8087/v2/servers/_defaultServer_/status", cs.Cs.Node)
	return url
}

func (cs *CatalogService) hasTag(tag Tag) bool {
	for _, t := range cs.Cs.ServiceTags {
		if t == tag.buildTag() {
			return true
		}
	}
	return false
}

func (cs *CatalogService) serviceRegister(c *api.Client) {
	reg := api.CatalogRegistration{
		Node:            cs.Cs.Node,
		Address:         cs.Cs.Address,
		Datacenter:      cs.Dc,
		TaggedAddresses: cs.Cs.TaggedAddresses,
		Service: &api.AgentService{
			ID:                cs.Cs.ServiceID,
			Service:           cs.Cs.ServiceName,
			Tags:              cs.Cs.ServiceTags,
			Port:              cs.Cs.ServicePort,
			Address:           cs.Cs.ServiceAddress,
			EnableTagOverride: true,
		},
	}
	c.Catalog().Register(&reg, nil)
	fmt.Printf("%s service for node %s registered with tags %s\n", cs.Cs.ServiceName, cs.Cs.Node, cs.Cs.ServiceTags)
}

func (cs *CatalogService) serviceAddTag(c *api.Client, s *api.CatalogService, tag Tag) error {
	if !cs.hasTag(tag) {
		cs.Cs.ServiceTags = append(cs.Cs.ServiceTags, tag.buildTag())
		cs.serviceRegister(c)
	}
	return nil
}

func (cs *CatalogService) serviceDeleteTag(c *api.Client, s *api.CatalogService, tag Tag) error {
	if !cs.hasTag(tag) {
		return nil
	}
	var tags []string
	for _, t := range cs.Cs.ServiceTags {
		if t != tag.buildTag() {
			tags = append(tags, t)
		}
	}
	cs.Cs.ServiceTags = tags
	cs.serviceRegister(c)

	return nil
}

func searchServiceWithoutTag(c []*api.CatalogService, unexpectedTag Tag) (api.CatalogService, error) {
	var ret api.CatalogService
	for _, s := range c {
		cs := CatalogService{Cs: s}
		if !cs.hasTag(unexpectedTag) {
			return *s, nil
		}
	}
	return ret, fmt.Errorf("Cannot found instance without tag %s", unexpectedTag)
}

func main() {
	transport := digest.NewTransport("admin", "admin.123")
	flag.Parse()
	var tag Tag
	if *tagOpts != "" {
		tag.deconstructTag(*tagOpts)
	}

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

	if *listActionOpts {
		for _, s := range catalogServices {
			cs := CatalogService{Dc: *datacenterName, Cs: s}
			if *tagOpts != "" && !cs.hasTag(tag) {
				continue
			}
			currentConnections, _ := lib.GetMetrics(cs.getURL(), transport)
			fmt.Printf("[%s] node:%s lan:%s wan:%s tags:%s current_connections:%d\n",
				s.ServiceName,
				s.Node,
				s.TaggedAddresses["lan"],
				s.TaggedAddresses["wan"],
				s.ServiceTags,
				currentConnections.CurrentConnections,
			)
		}
		os.Exit(0)
	} else if *addTagActionOpts {
		service, err := searchServiceWithoutTag(catalogServices, tag)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		cs := CatalogService{Dc: *datacenterName, Cs: &service}
		err = cs.serviceAddTag(client, &service, tag)
		if err != nil {
		}
	} else if *deleteTagActionOpts {
		for _, service := range catalogServices {
			cs := CatalogService{Dc: *datacenterName, Cs: service}
			cs.serviceDeleteTag(client, service, tag)
		}
	} else {
		flag.Usage()
	}
}
