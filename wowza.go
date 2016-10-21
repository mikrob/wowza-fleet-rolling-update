package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
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
	cs *api.CatalogService
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (cs *CatalogService) overrideTagService(c *api.Client, s *api.CatalogService, tag Tag) error {
	if !hasTag(s.ServiceTags, tag.buildTag()) {
		s.ServiceTags = append(s.ServiceTags, tag.buildTag())
	}

	return nil
}

func searchService(c []*api.CatalogService, unexpectedTag Tag) (api.CatalogService, error) {
	var ret api.CatalogService
	for _, s := range c {
		if !hasTag(s.ServiceTags, unexpectedTag.buildTag()) {
			return *s, nil
		}
	}
	return ret, fmt.Errorf("Cannot found instance without tag %s", unexpectedTag)
}

func main() {
	serviceName := flag.String("service", "", "Consul service name")
	datacenterName := flag.String("dc", "", "Consul datacenter")
	tagOpts := flag.String("tag", "", "Tag (key=value)")
	flag.Parse()

	var tag Tag
	tag.deconstructTag(*tagOpts)

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
	var cs CatalogService

	// search a service member without tag "should_version"
	service, err := searchService(catalogServices, tag)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	cs.cs = &service
	err = cs.overrideTagService(client, &service, tag)
	if err != nil {

	}
	fmt.Println(service.ServiceTags)

}
