Nexenta Plugin for Docker Volumes
======================================


## Description
This plugin provides the ability to use Nexenta Storage Clusters as backend
devices in a Docker environment.

## Prerequisites
### Golang
To install the latest Golang, just follow the easy steps on the Go install
websiste.  Again, I prefer to download the tarball and install myself rather
than use a package manager like apt or yum:
[Get Go](https://golang.org/doc/install)

NOTE:
It's very important that you follow the directions and setup your Go
environment.  After downloading the appropriate package be sure to scroll down
to the Linux, Mac OS X, and FreeBSD tarballs section and set up your Go
environment as per the instructions.

### GCC
```
  apt-get install gcc
```
NOTE:
Should be run as root and command may differ depending on your OS. 

### Docker
You can find instructions and steps on the Docker website here:
[Get Docker](https://docs.docker.com/linux/step_one/)

### Nexenta
For NS5 NFS a pool and parent filesystem must be precreated on NS appliance.

### NFS
If you are using NexentaStor5 as a backend via NFS IOProtocol, you need nfs-common package installed.
```
  apt-get install nfs-common
```
NOTE:
Should be run as root and command may differ depending on your OS.

## Configuration
Example config file can be found here:
  ```
  https://github.com/Nexenta/nexenta-docker-driver/blob/master/nvd.json
  ```
  
Default path to config file is
  ```
  /etc/nvd/nvd.json
  ```

## Driver Installation
After the above Prerequisites are met, clone repository and use the Makefile:
  ```
  git clone https://github.com/nexenta/nexenta-docker-driver
  make
  ```

In addition to providing the source, this should also build and install the
nvd binary in your Golang bin directory.

You will need to make sure you've added the $GOPATH/bin to your path,
AND on Ubuntu you will also need to enable the use of the GO Bin path by sudo;
either run visudo and edit, or provide an alias in your .bashrc file.

You need to pre-create a folder for the GO code.
For example in your .bashrc set the following alias after setting up PATH:
  ```
  export GOPATH=<your GO folder>
  export PATH=$PATH:/usr/local/go/bin:<your GO folder>/bin/
  alias sudo='sudo env PATH=$PATH'
  ```

## Starting the daemon
After install and setting up a configuration, all you need to is start the
nexenta-docker-driver daemon so tha it can accept requests from Docker.

  ```
  sudo nvd daemon start -v
  ```

## Usage Examples
Now that the daemon is running, you're ready to issue calls via the Docker
Volume API and have the requests serviced by the Nexenta Driver.

For a list of avaialable commands run:
  ```
  docker volume --help
  ```

Here's an example of how to create a Nexenta volume using the Docker Volume
API:
  ```
  docker volume create -d nvd --name=testvolume
  ```

Now in order to use that volume with a Container you simply specify
  ```
  docker run -v testvolume:/Data --volume-driver=nvd -i -t ubuntu
  /bin/bash
  ```

Note that if you had NOT created the volume already, Docker will issue the
create call to the driver for you while launching the container.  The Driver
create method checks the Nexenta backend to see if the Volume already exists,
if it does it just passes back the info for the existing volume, otherwise it
runs through the create process and creates the Volume on the Nexenta
backend.
