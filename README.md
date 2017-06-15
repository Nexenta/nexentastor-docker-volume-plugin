Nexenta Plugin for Docker Volumes
======================================

Usage:
1) Clone this repository
```
git clone https://github.com/nexenta/nexenta-docker-driver && cd nexenta-docker-driver
```
2) Copy nvd.json.example to /etc/nvd/nvd.json and change values according to your NexentaStor setup
```
mkdir /etc/nvd
cp nvd.json.example /etc/nvd/nvd.json
```
3) Install and run the plugin
```
make
```
4) Use plugin to create docker volumes
```
docker volume create -d nexenta/nexentastor-nfs-plugin --name=testvolume
docker run -v testvolume:/Data -it ubuntu /bin/bash
```
