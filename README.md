# Docker Volume Plugin for NexentaStor

[![Build Status](https://travis-ci.org/Nexenta/nexentastor-docker-volume-plugin.svg?branch=master)](https://travis-ci.org/Nexenta/nexentastor-docker-volume-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/Nexenta/nexentastor-docker-volume-plugin)](https://goreportcard.com/report/github.com/Nexenta/nexentastor-docker-volume-plugin)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)

This is **development** repository,
stable versions are published on
[DockerHub plugin page](https://hub.docker.com/r/nexenta/nexentastor-docker-volume-plugin/tags).

NexentaStor product page: [https://nexenta.com/products/nexentastor](https://nexenta.com/products/nexentastor).

## Supported versions

|                | NexentaStor 5.1.2                                                               | NexentaStor 5.2.0                                                               | NexentaStor 5.2.1                                                               |
|----------------|---------------------------------------------------------------------------------|---------------------------------------------------------------------------------|---------------------------------------------------------------------------------|
| Docker >=17.06 | [1.X.X](https://hub.docker.com/r/nexenta/nexentastor-docker-volume-plugin/tags) | [1.X.X](https://hub.docker.com/r/nexenta/nexentastor-docker-volume-plugin/tags) | [1.X.X](https://hub.docker.com/r/nexenta/nexentastor-docker-volume-plugin/tags) |

## Features

- Create new volume
- Use existing volume
- NFS mount protocol

## Requirements

Following utilities must be installed on Docker setup for NFS mounts:
```bash
apt install -y nfs-common
```

## Installation

1. Create NexentaStor dataset for the volume plugin, example: `spool01/dataset`.
   Volume plugin will create filesystems in this dataset and mount them to use as Docker volumes.
2. Create plugin configuration file: `/etc/nexentastor-docker-volume-plugin/config.yaml`. Plugin configuration
    [example](https://github.com/Nexenta/nexentastor-docker-volume-plugin/blob/master/etc/nexentastor-docker-volume-plugin/config.yaml):
   ```yaml
   restIp: https://10.3.3.4:8443,https://10.3.3.5:8443 # [required] NexentaStor REST API endpoint(s)
   username: admin                                     # [required] NexentaStor REST API username
   password: p@ssword                                  # [required] NexentaStor REST API password
   defaultDataset: spool01/dataset                     # [required] dataset to use ('pool/dataset')
   defaultDataIp: 20.20.20.21                          # [required] data IP or HA VIP
   #defaultMountOptions: noatime                       # mount options (mount -o ...)
   #debug: true                                        # more logs (true/false)
   ```
3. Install volume plugin:
   ```
   docker plugin install nexenta/nexentastor-docker-volume-plugin
   ```
4. Enable volume plugin:
   ```
   docker plugin enable nexenta/nexentastor-docker-volume-plugin
   ```

Volume plugin should be listed after installation:

```
$ docker plugin list
ID             NAME                                              DESCRIPTION                            ENABLED
b227326b403d   nexenta/nexentastor-docker-volume-plugin:latest   Docker Volume Plugin for NexentaStor   true
```

## Configuration

All plugin configuration options:

| Name                  | Description                                                     | Required | Example                 |
|-----------------------|-----------------------------------------------------------------|----------|-------------------------|
| `restIp`              | NexentaStor REST API endpoint(s); `,` to separate cluster nodes | yes      | `https://10.3.3.4:8443` |
| `username`            | NexentaStor REST API username                                   | yes      | `admin`                 |
| `password`            | NexentaStor REST API password                                   | yes      | `p@ssword`              |
| `defaultDataset`      | parent dataset for plugin's filesystems ("pool/dataset")        | yes      | `spool01/dataset`       |
| `defaultDataIp`       | NexentaStor data IP or HA VIP for mounting shares               | yes      | `20.20.20.21`           |
| `defaultMountOptions` | NFS mount options: `mount -o ...`<br>(default: "")              | no       | `noatime,nosuid`        |
| `debug`               | print more logs (default: false)                                | no       | `true`                  |

**Note**: parameter `restIp` can point on a single NexentaStor appliance or on each of the nodes of HA cluster.

## Usage

- List all existing volumes.
   All NexentaStor filesystems under configured `defaultDataset` path will be already listed there as Docker volumes.
   ```bash
   docker volume list
   ```
- Create Docker volume `testvolume` if NexentaStor filesystem doesn't exist:
   ```bash
   docker volume create -d nexenta/nexentastor-docker-volume-plugin --name=testvolume
   ```
   **Note**: This operation will create a filesystem on NexentaStore in case it doesn't exist.
- Run container which uses created volume `testvolume`:
   ```bash
   docker run -v testvolume:/data -it --rm ubuntu /bin/bash
   ```
   **Note**: This operation will share filesystem and mount it.
- Remove Docker volume command doesn't remove any filesystem from NexentaStore and doesn't affect Docker volumes list.

## Uninstall

```bash
# disable plugin
docker plugin disable nexenta/nexentastor-docker-volume-plugin

# remove plugin
docker plugin remove nexenta/nexentastor-docker-volume-plugin
```

## Knows Issues

- Creating volumes during NS HA failover might prevent the failover.

## Troubleshooting

- Plugin logs
  ```bash
  # log file
  tail -f /var/lib/docker/plugins/*/rootfs/var/log/nexentastor-docker-volume-plugin.log

  # system journal
  journalctl -f -u docker.service
  ```
- Check mounts exist on host
  ```bash
  mount | grep /var/lib/docker/plugins
  mount | grep NEXENTASTOR_DATA_IP_FROM_CONFIG_FILE
  ```
- Show installed plugins:
  ```bash
  docker plugin list
  ```

## Development

Commits should follow [Conventional Commits Spec](https://conventionalcommits.org).

### Build

Build and push commands take Git branch as a plugin version to build.

```bash
# Set environment variable for any of these commands to over the version
# The default version for make commands is the current Git branch.
# VERSION=1.0.0 make ...

# Note: same operations work with "*-production" postfix.

# print variables and help
make

# build with development tag (default for `make` w/o params)
make build-development

# enable development version of plugin on local Docker setup
make enable-development

# disable and delete development version of plugin
make uninstall-development

# publish the latest built container to the local registry (see `Makefile`)
make push-development

# update deps
# go get -u github.com/golang/dep/cmd/dep
~/go/bin/dep ensure
```

### Test
```bash
# Test options:
# - TEST_DOCKER_IP=10.3.199.249 # Docker setup IP address to test on
# - NOCOLORS=true               # disable colors

# run all tests using local Docker registry:
TEST_DOCKER_IP=10.3.199.249 make test-e2e-docker-development

# run all tests using local Docker registry (in container):
TEST_DOCKER_IP=10.3.199.249 make test-e2e-docker-development-container

# configure Docker to trust insecure registries:
# add `{"insecure-registries":["10.3.199.92:5000"]}` to:
vim /etc/docker/daemon.json
service docker restart
```

### Debug

Send requests to the plugin:
```bash
# plugin container id can be found in `journalctl -f -u docker.service` output
curl -X POST \
    -d '{}' \
    -H "Content-Type: application/json" \
    --unix-socket /run/docker/plugins/%ID%/nsdvp.sock \
    http://localhost/VolumeDriver.List

# check built version
./plugin/rootfs/bin/nexentastor-docker-volume-plugin --version
```

### Release

All development happens in `master` branch,
when it's time to publish a new version,
new git tag should be created.

1. Build and [test](#test) the new version using local registry:
   ```bash
   # build development version:
   make build-development
   # publish to local registry
   make push-development
   # run test commands
   ```

2. To release a new version run command:
   ```bash
   VERSION=X.X.X make release
   ```
   This script does following:
   - generates new `CHANGELOG.md`
   - builds plugin version 'nexenta/nexentastor-docker-volume-plugin:X.X.X'
   - logs in to hub.docker.com
   - publishes plugin version 'nexenta/nexentastor-docker-volume-plugin:X.X.X' to hub.docker.com
   - creates git tag 'vX.X.X' and pushes it to the repository
   - asks to update 'latest' tag on hub.docker.com, updates it if needed.

   **Note**: Release command does this, but `latest` tag can be built and pushed later manually if needed.
   This command takes the most recent built plugin (from local `./plugin` folder)
   and pushes it as `latest` tag to hub.docker.com.
   ```
   make update-latest
   ```

3. Update Github [releases](https://github.com/Nexenta/nexentastor-docker-volume-plugin/releases).

4. Update Docker Hub
   [description](https://cloud.docker.com/u/nexenta/repository/docker/nexenta/nexentastor-docker-volume-plugin/general)
   if needed.
