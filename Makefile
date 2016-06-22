NVD_EXE = nvd
FLAGS = -v

all: $(NVD_EXE)

GO_FILES = src/github.com/Nexenta/nexenta-docker-driver/nvd/nvd.go \
	src/github.com/Nexenta/nexenta-docker-driver/nvd/nvdcli/nvdcli.go \
	src/github.com/Nexenta/nexenta-docker-driver/nvd/nvdcli/daemoncli.go \
	src/github.com/Nexenta/nexenta-docker-driver/nvd/nvdcli/volumecli.go \
	src/github.com/Nexenta/nexenta-docker-driver/nvd/daemon/daemon.go \
	src/github.com/Nexenta/nexenta-docker-driver/nvd/daemon/driver.go \
	src/github.com/Nexenta/nexenta-docker-driver/nvd/nvdapi/nvdapi.go

$(GO_FILES): setup

deps: setup
	GOPATH=$(shell pwd) go get github.com/docker/go-plugins-helpers/volume
	GOPATH=$(shell pwd) go get github.com/codegangsta/cli
	GOPATH=$(shell pwd) go get github.com/Sirupsen/logrus
	GOPATH=$(shell pwd) go get github.com/coreos/go-systemd/util
	GOPATH=$(shell pwd) go get github.com/opencontainers/runc/libcontainer/user
	GOPATH=$(shell pwd) go get golang.org/x/net/proxy


$(NVD_EXE): $(GO_FILES)
	GOPATH=$(shell pwd) go install github.com/Nexenta/nexenta-docker-driver/nvd

build:
	GOPATH=$(shell pwd) go build $(FLAGS) github.com/Nexenta/nexenta-docker-driver/nvd

setup: 
	mkdir -p src/github.com/Nexenta/nexenta-docker-driver/ 
	cp -R nvd/ src/github.com/Nexenta/nexenta-docker-driver/nvd 

lint:
	GOPATH=$(shell pwd) go get -v github.com/golang/lint/golint
	for file in $$(find . -name '*.go' | grep -v vendor | grep -v '\.pb\.go' | grep -v '\.pb\.gw\.go'); do \
		golint $${file}; \
		if [ -n "$$(golint $${file})" ]; then \
			exit 1; \
		fi; \
	done

clean:
	GOPATH=$(shell pwd) go clean


clobber:
	rm -rf src/github.com/Nexenta/nexenta-docker-driver
	rm -rf bin/ pkg/

