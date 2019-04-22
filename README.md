# NexentaStor Volume Driver for Docker

[![Build Status](https://travis-ci.org/Nexenta/nexenta-docker-driver.svg?branch=master)](https://travis-ci.org/Nexenta/nexenta-docker-driver)
[![Go Report Card](https://goreportcard.com/badge/github.com/Nexenta/nexenta-docker-driver)](https://goreportcard.com/report/github.com/Nexenta/nexenta-docker-driver)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)

This is **development** repository,
stable versions are published on [DockerHub driver page](https://hub.docker.com/r/nexenta/nexentastor-nfs-plugin/tags).

NexentaStor product page: [https://nexenta.com/products/nexentastor](https://nexenta.com/products/nexentastor).

## Supported versions

|                | NexentaStor 5.1                                                       | NexentaStor 5.2                                                       |
|----------------|-----------------------------------------------------------------------|-----------------------------------------------------------------------|
| Docker >=17.06 | [1.0.0](https://hub.docker.com/r/nexenta/nexentastor-nfs-plugin/tags) | [1.0.0](https://hub.docker.com/r/nexenta/nexentastor-nfs-plugin/tags) |

## Requirements

- Following utilities must be installed on Docker setup:
  ```bash
  # for NFS mounts
  apt install -y nfs-common
  ```

## Installation

1. Create NexentaStor dataset for the volume driver, example: `spool01/dataset`.
   Volume driver will create filesystems in this dataset and mount them to use as Docker volumes.
2. Create driver configuration file: `/etc/nvd/nvd.yaml`. Driver configuration
    [example](/etc/nvd/nvd.yaml):
   ```yaml
   restIp: https://10.3.3.4:8443,https://10.3.3.5:8443 # [required] NexentaStor REST API endpoint(s)
   username: admin                                     # [required] NexentaStor REST API username
   password: p@ssword                                  # [required] NexentaStor REST API password
   defaultDataset: spool01/dataset                     # [required] dataset to use ('pool/dataset')
   defaultDataIp: 20.20.20.21                          # [required] data IP or HA VIP
   #defaultMountOptions: noatime                       # mount options (mount -o ...)
   #debug: true                                        # more logs (true/false)
   ```

   All driver configuration options:

   | Name                  | Description                                                     | Required | Example                 |
   |-----------------------|-----------------------------------------------------------------|----------|-------------------------|
   | `restIp`              | NexentaStor REST API endpoint(s); `,` to separate cluster nodes | yes      | `https://10.3.3.4:8443` |
   | `username`            | NexentaStor REST API username                                   | yes      | `admin`                 |
   | `password`            | NexentaStor REST API password                                   | yes      | `p@ssword`              |
   | `defaultDataset`      | parent dataset for driver's filesystems ("pool/dataset")        | yes      | `spool01/dataset`       |
   | `defaultDataIp`       | NexentaStor data IP or HA VIP for mounting shares               | yes      | `20.20.20.21`           |
   | `defaultMountOptions` | NFS mount options: `mount -o ...`<br>(default: "")              | no       | `noatime,nosuid`        |
   | `debug`               | print more logs (default: false)                                | no       | `true`                  |

   **Note**: parameter `restIp` can point on a single NexentaStor appliance or on each of the nodes of HA cluster.

3. Install volume driver:
   ```
   docker plugin install nexenta/nexentastor-nfs-plugin:1.0.0
   ```
4. Enable volume driver:
   ```
   docker plugin enable nexenta/nexentastor-nfs-plugin:1.0.0
   ```

Volume driver should be listed after installation:

```
$ docker plugin list
ID             NAME                                   DESCRIPTION                            ENABLED
b227326b403d   nexenta/nexentastor-nfs-plugin:1.0.0   NexentaStor Volume Driver for Docker   true
```

## Usage

- List all existing volumes.
   All NexentaStor filesystems under configured `defaultDataset` path will be already listed there as Docker volumes.
   ```bash
   docker volume list
   ```
- Create Docker volume `testvolume` if NexentaStor filesystem doesn't exist:
   ```bash
   docker volume create -d nexenta/nexentastor-nfs-plugin:1.0.0 --name=testvolume
   ```
   **Note**: This operation will create filesystem on NexentaStore.
- Run container which uses created volume `testvolume`:
   ```bash
   docker run -v testvolume:/data -it --rm ubuntu /bin/bash
   ```
   **Note**: This operation will share filesystem and mount it.
- Remove Docker volume command doesn't remove any filesystem from NexentaStore and doesn't affect Docker volumes list.

## Uninstall

```bash
# disable driver
docker plugin disable nexenta/nexentastor-nfs-plugin:1.0.0

# remove driver
docker plugin remove nexenta/nexentastor-nfs-plugin:1.0.0
```

## Development

Commits should follow [Conventional Commits Spec](https://conventionalcommits.org).

### Build

Build and push commands take Git branch as a plugin version to build.

```bash
# build with development tag (default for `make` w/o params)
make build-development

# build with production tag
make build-production

# update deps
# go get -u github.com/golang/dep/cmd/dep
~/go/bin/dep ensure

# check built version
./plugin/rootfs/bin/nvd --version
```

### Debug

Send requests to the driver:
```bash
# driver container id can be found in `journalctl -f -u docker.service` output
curl -X POST \
    -d '{}' \
    -H "Content-Type: application/json" \
    --unix-socket /run/docker/plugins/%ID%/nvd.sock \
    http://localhost/VolumeDriver.List
```

### Publish

```bash
# push the latest built container to the local registry (see `Makefile`)
make push-development

# push the latest built container to hub.docker.com
make push-production
```

### Release

All development happens in `master` branch,
when it's time to publish a new version,
new git tag should be created.

1. Build and test the new version using local registry:
   ```bash
   # build development version:
   make build-development
   # publish to local registry
   make push-development
   # for now, manually test plugin using local registry
   ```

2. Release a new version. This script does following:
   - generates new `CHANGELOG.md`
   - builds plugin version 'nexenta/nexentastor-nfs-plugin:X.X.X'
   - logs in to hub.docker.com
   - publishes plugin version 'nexenta/nexentastor-nfs-plugin:X.X.X' to hub.docker.com
   - creates git tag 'X.X.X' and pushes it to the repository
```bash
VERSION=X.X.X make release
```

3. Update Github [releases](https://github.com/Nexenta/nexenta-docker-driver/releases).

## Troubleshooting

- Show installed drivers:
  ```bash
  docker plugin list
  ```
- Driver logs
  ```bash
  # log file
  tail -f /var/lib/docker/plugins/*/rootfs/var/log/nvd.log

  # system journal
  journalctl -f -u docker.service
  ```
- Check mounts exist on host
  ```bash
  mount | grep /var/lib/docker/plugins
  mount | grep NEXENTASTOR_DATA_IP_FROM_CONFIG_FILE
  ```
- Configure Docker to trust insecure registries:
  ```bash
  # add `{"insecure-registries":["10.3.199.92:5000"]}` to:
  vim /etc/docker/daemon.json
  service docker restart
  ```
