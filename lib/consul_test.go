package lib

import (
	"fmt"
	"testing"

	api "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
)

type configCallback func(c *api.Config)

func makeClient(t *testing.T) (*api.Client, *testutil.TestServer) {
	return makeClientWithConfig(t, nil, nil)
}

func makeClientWithConfig(t *testing.T, cb1 configCallback, cb2 testutil.ServerConfigCallback) (*api.Client, *testutil.TestServer) {

	// Make client config
	conf := api.DefaultConfig()
	//conf.Datacenter = "dc1wowzatest"
	if cb1 != nil {
		//t.Log("CB1 is NOT NIL")
		cb1(conf)
	}

	// Create server

	server := testutil.NewTestServerConfig(t, cb2)
	conf.Address = server.HTTPAddr

	// Create client
	client, err := api.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	return client, server
}

func TestBuildTagShouldReturnWellFormatedString(t *testing.T) {
	tag := Tag{Key: "toto", Value: "titi"}
	result, _ := tag.BuildTag()
	if result != "toto=titi" {
		t.Error("Build Tag shoudl return toto=titi and it returned", result)
	}
}

func TestBuildTagShouldNotBuildEmptyTag(t *testing.T) {
	tag := Tag{}
	_, err := tag.BuildTag()
	if err == nil {
		t.Error("Should have raise an error because try to build an empty tag")
	}
}

func TestDesconstructTagShouldBuildAGoodTagFromString(t *testing.T) {
	tagStr := "toto=titi"
	//var tag *Tag = &Tag{}
	tag := &Tag{}
	tag.DeconstructTag(tagStr)
	if tag.Key != "toto" {
		t.Error("Key should be toto and is : ", tag.Key)
	}
	if tag.Value != "titi" {
		t.Error("Value should be titi and is : ", tag.Value)
	}
}

func registerFakeWowzaService(serviceName string, nodeName string, ip string, client *api.Client, t *testing.T) {
	catalog := client.Catalog()
	service := &api.AgentService{
		ID:      serviceName,
		Service: serviceName,
		Tags:    []string{"master=toto", "v1"},
		Port:    1935,
	}
	reg := &api.CatalogRegistration{
		Datacenter: "dc1",
		Node:       nodeName,
		Address:    ip,
		Service:    service,
		Check:      nil,
	}
	catalog.Register(reg, nil)
	testutil.WaitForResult(func() (bool, error) {
		if _, err := catalog.Register(reg, nil); err != nil {
			return false, err
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func initializeConsul(t *testing.T) (*api.Catalog, *testutil.TestServer, *api.Client) {
	t.Parallel()
	client, server := makeClient(t)

	registerFakeWowzaService("wowza-edge", "node1", "192.168.1.1", client, t)
	registerFakeWowzaService("wowza-edge", "node2", "192.168.1.2", client, t)
	registerFakeWowzaService("wowza-edge", "node3", "192.168.1.3", client, t)
	registerFakeWowzaService("wowza-edge", "node4", "192.168.1.4", client, t)
	catalog := client.Catalog()
	return catalog, server, client
}

func TestGetUrlShouldReturnRightUrl(t *testing.T) {
	catalog, server, _ := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retriving services")
	}

	for _, s := range catalogServices {
		cs := CatalogService{Cs: s}
		if cs.GetURL() != fmt.Sprintf("http://%s.botsunit.io:8087/v2/servers/_defaultServer_/status", cs.Cs.Node) {
			t.Error("Get URL return is malformated")
		}
	}
}

func TestHasTagReturnTrueIfServiceHasTag(t *testing.T) {
	catalog, server, _ := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}
	cs := CatalogService{Cs: catalogServices[0]}
	tag := Tag{
		Key:   "master",
		Value: "toto",
	}
	if !cs.HasTag(tag) {
		t.Error("Service should have tag master=toto")
	}
}

func TestHasTagReturnFalseIfServiceHasNotTag(t *testing.T) {
	catalog, server, _ := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}
	cs := CatalogService{Cs: catalogServices[0]}
	tag := Tag{
		Key:   "tutu",
		Value: "tata",
	}
	if cs.HasTag(tag) {
		t.Error("Service should NOT have tag tutu=tata")
	}
}

func TestServiceAddTag(t *testing.T) {
	catalog, server, client := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}
	cs := CatalogService{Cs: catalogServices[0]}
	tag := Tag{
		Key:   "maintenance",
		Value: "true",
	}
	cs.ServiceAddTag(client, catalogServices[0], tag)

	if !cs.HasTag(tag) {
		t.Errorf("Service should have new added tag : key %s, value %s", tag.Key, tag.Value)
	}
}

func TestServiceDeleteTag(t *testing.T) {
	catalog, server, client := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}
	cs := CatalogService{Cs: catalogServices[0]}
	tag := Tag{
		Key:   "master",
		Value: "toto",
	}
	cs.ServiceDeleteTag(client, catalogServices[0], tag)

	if cs.HasTag(tag) {
		t.Errorf("Service should NOT have tag : key %s, value %s because delete has been called", tag.Key, tag.Value)
	}
}

func TestServiceDeleteTagUnexisting(t *testing.T) {
	catalog, server, client := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}
	cs := CatalogService{Cs: catalogServices[0]}
	tag := Tag{
		Key:   "hibou",
		Value: "caillou",
	}
	cs.ServiceDeleteTag(client, catalogServices[0], tag)

	if cs.HasTag(tag) {
		t.Errorf("Service should NOT have tag : key %s, value %s because delete has been called", tag.Key, tag.Value)
	}
}

func TestSearchServiceWithoutTag(t *testing.T) {
	catalog, server, _ := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}

	tag := Tag{
		Key:   "maintenance",
		Value: "true",
	}

	cs, err := SearchServiceWithoutTag(catalogServices, tag)
	if err != nil {
		t.Error("Error while searching service with no tag maintenanance=true")
	}
	if cs.Node == "" {
		t.Error("Didnt found service with tag maintenance=true")
	}
}

func TestSearchServiceWithoutTagButAllHaveTag(t *testing.T) {
	catalog, server, _ := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retrieving services")
	}

	tag := Tag{
		Key:   "master",
		Value: "toto",
	}

	cs, err := SearchServiceWithoutTag(catalogServices, tag)
	if err == nil {
		t.Error("Should make error because all service has searched tag")
	}
	if cs.Node != "" {
		t.Error("Should be empty because all services has searched tag")
	}
}
