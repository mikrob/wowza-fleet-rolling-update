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
	result := tag.BuildTag()
	if result != "toto=titi" {
		t.Error("Build Tag shoudl return toto=titi and it returned", result)
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
		Tags:    []string{"master", "v1"},
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

func initializeConsul(t *testing.T) (*api.Catalog, *testutil.TestServer) {
	t.Parallel()
	client, server := makeClient(t)

	registerFakeWowzaService("wowza-edge", "node1", "192.168.1.1", client, t)
	registerFakeWowzaService("wowza-edge", "node2", "192.168.1.2", client, t)
	registerFakeWowzaService("wowza-edge", "node3", "192.168.1.3", client, t)
	registerFakeWowzaService("wowza-edge", "node4", "192.168.1.4", client, t)
	catalog := client.Catalog()
	return catalog, server
}

func TestGetUrlShouldReturnRightUrl(t *testing.T) {
	catalog, server := initializeConsul(t)
	defer server.Stop()
	catalogServices, _, err := catalog.Service("wowza-edge", "", nil)
	if err != nil {
		t.Error("Error while retriving services")
	}

	for _, s := range catalogServices {
		cs := CatalogService{Cs: s}
		t.Log("CATALOG SERVICE :")
		t.Logf("%+v\n", cs.Cs)
		if cs.GetURL() != fmt.Sprintf("http://%s.botsunit.io:8087/v2/servers/_defaultServer_/status", cs.Cs.Node) {
			t.Error("Get URL return is malformated")
		}
	}

	// testutil.WaitForResult(func() (bool, error) {
	// 	datacenters, err := catalog.Datacenters()
	// 	if err != nil {
	// 		return false, err
	// 	}
	//
	// 	if len(datacenters) == 0 {
	// 		return false, fmt.Errorf("Bad: %v", datacenters)
	// 	}
	// 	t.Log(datacenters)
	// 	return true, nil
	// }, func(err error) {
	// 	t.Fatalf("err: %s", err)
	// })
}
