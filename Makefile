NED_EXE = nedv
FLAGS = -v

all: $(NED_EXE)

GO_FILES = src/github.com/Nexenta/nedge-docker-volume/nedv/nedv.go \
	src/github.com/Nexenta/nedge-docker-volume/nedv/nedcli/nedcli.go \
	src/github.com/Nexenta/nedge-docker-volume/nedv/nedcli/Foo.go \
	src/github.com/Nexenta/nedge-docker-volume/nedv/nedcli/Bar.go \
	src/github.com/Nexenta/nedge-docker-volume/nedv/daemon/daemon.go \
	src/github.com/Nexenta/nedge-docker-volume/nedv/daemon/driver.go \
	src/github.com/Nexenta/nedge-docker-volume/nedv/nedapi/nedapi.go

$(GO_FILES): setup

deps: setup
	GOPATH=$(shell pwd) go get github.com/docker/go-plugins-helpers/volume
	GOPATH=$(shell pwd) go get github.com/codegangsta/cli
	GOPATH=$(shell pwd) go get github.com/Sirupsen/logrus
	GOPATH=$(shell pwd) go get github.com/coreos/go-systemd/util
	GOPATH=$(shell pwd) go get github.com/opencontainers/runc/libcontainer/user
	GOPATH=$(shell pwd) go get golang.org/x/net/proxy


$(NED_EXE): $(GO_FILES)
	GOPATH=$(shell pwd) go install github.com/Nexenta/nedge-docker-volume/nedv

build:
	GOPATH=$(shell pwd) go build $(FLAGS) github.com/Nexenta/nedge-docker-volume/nedv

setup: 
	mkdir -p src/github.com/Nexenta/nedge-docker-volume/ 
	cp -R ned/ src/github.com/Nexenta/nedge-docker-volume/nedv 

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
	rm -rf src/github.com/Nexenta/nedge-docker-volume
	rm -rf bin/ pkg/

