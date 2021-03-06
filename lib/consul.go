package lib

import (
	"errors"
	"fmt"
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

// BuildTag build a tag string from a tag struct
func (t Tag) BuildTag() (string, error) {
	var err error
	if t.Key == "" || t.Value == "" {
		err = errors.New("Should not build tag with empty key or value")
		fmt.Println("Error while building tag :", err.Error())
	}
	if err != nil {
		return "", err
	} else {
		return fmt.Sprintf("%s=%s", t.Key, t.Value), err
	}
}

//DeconstructTag allow to construct a tag from a given string containing key/value separated with =
func (t *Tag) DeconstructTag(tag string) {
	k, v := split(tag, "=")
	t.Key = k
	t.Value = v
}

// CatalogService extends api.CatalogService
type CatalogService struct {
	Cs *api.CatalogService
	Dc string
}

// GetURL build and url from given CatalogService
func (cs *CatalogService) GetURL() string {
	url := fmt.Sprintf("http://%s.botsunit.io:8087/v2/servers/_defaultServer_/status", cs.Cs.Node)
	return url
}

// HasTag check if catalog service has a given tag
func (cs *CatalogService) HasTag(tag Tag) bool {
	for _, t := range cs.Cs.ServiceTags {
		newtag, _ := tag.BuildTag()
		if t == newtag {
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

//ServiceAddTag allow to add a tag on a service
func (cs *CatalogService) ServiceAddTag(c *api.Client, s *api.CatalogService, tag Tag) error {
	if !cs.HasTag(tag) {
		fmt.Println("ADD TAG : ", tag.Key)
		newTag, err := tag.BuildTag()
		if err != nil {
			fmt.Println("Error while building tag to add to service : ", err.Error())
			panic(err.Error())
		}
		cs.Cs.ServiceTags = append(cs.Cs.ServiceTags, newTag)
		cs.serviceRegister(c)
	}
	return nil
}

//ServiceDeleteTag allow to delete a tag on a service
func (cs *CatalogService) ServiceDeleteTag(c *api.Client, s *api.CatalogService, tag Tag) error {
	if !cs.HasTag(tag) {
		return nil
	}
	var tags []string
	for _, t := range cs.Cs.ServiceTags {
		newtag, err := tag.BuildTag()
		if err != nil {
			fmt.Println("Error while building tag to add to service : ", err.Error())
			panic(err.Error())
		}
		if t != newtag {
			tags = append(tags, t)
		}
	}
	cs.Cs.ServiceTags = tags
	cs.serviceRegister(c)

	return nil
}

//SearchServiceWithoutTag allow to search a service without a given tag, return the first that doesn't have this tag
func SearchServiceWithoutTag(c []*api.CatalogService, unexpectedTag Tag) (api.CatalogService, error) {
	var ret api.CatalogService
	for _, s := range c {
		cs := CatalogService{Cs: s}
		if !cs.HasTag(unexpectedTag) {
			return *s, nil
		}
	}
	return ret, fmt.Errorf("Cannot found instance without tag %s", unexpectedTag)
}

//SearchServiceWithTag allow to search a service with a given tag, return the first that do have this tag
func SearchServiceWithTag(c []*api.CatalogService, expectedTag Tag) (api.CatalogService, error) {
	var ret api.CatalogService
	for _, s := range c {
		cs := CatalogService{Cs: s}
		if cs.HasTag(expectedTag) {
			return *s, nil
		}
	}
	return ret, fmt.Errorf("Cannot found instance with tag %s", expectedTag)
}
