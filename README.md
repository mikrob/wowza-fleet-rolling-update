# wowza-rolling-update

[![Build Status](https://jenkins.botsunit.com/jenkins/buildStatus/icon?job=wowza_rolling_update_test)](https://jenkins.botsunit.com/jenkins/job/wowza_rolling_update_test/)

`wowza-rolling-update` upgrades Wowza origin and edge containers with a rolling update process. Containers are destroyed and recreated with new unit file when no connections are found in Wowza container.

The update process can be describe steps by steps:

- you have to edit and commit fleet unit files for wowza-origin/edge by modifying the image tag used to start the container
- launch wowza-rolling-update with parameters `-update image:tag` representing the new image tag containers should run on
- wowza-rolling-update searches Consul service with a different `image=` tag value which is the running version of the container,
- tag one of the to-update container with tag `update=image:tag`,
- wait that wowza returns no connection to this container,
- destroy the unit,
- start the unit with the newly modified and committed unit file during step one,
- search again for outdated containers

## Requirements

- SSH access to one of the CoreOS fleet server
- Consul registrator is used in CoreOS to dynamicaly register and tags containers as Consul service (https://github.com/botsunit/registrator)
- Consul
- Wowza unit files ( https://gitlab.botsunit.com/infra/ansible_coreos/tree/master/services/wowza)

Consul services managed by `registrator` have to be registered with `EnableTagOverride: true`. Start containers with following environment variable:

```
docker run [...] -e "SERVICE_enable_tag_override=true" [...]
```

## Usage

List service nodes:

```
wowza-rolling-update -dc dc1streamingdev -service wowza-origin -list
```

Update fleet units for a given service:
```
wowza-rolling-update -dc dc1streamingdev -service wowza-origin -update eu.gcr.io/scalezen/wowza_bundle:0.3.4 -fleet-ssh-server coreosdev0001.botsunit.io -units-dir /Users/bjo/infra/ansible_coreos/services/wowza
```

You can also tag manually a Consul service node:

```
wowza-rolling-update -dc dc1streamingdev -service wowza-origin -add-tag -tag foo=bar
```

or delete a specific tag for all service nodes:

```
wowza-rolling-update -dc dc1streamingdev -service wowza-origin -delete-tag -tag foo=bar
```
