# NexentaStor Docker Volume Driver makefile

DRIVER_NAME = nexentastor-nfs-plugin
IMAGE_NAME ?= ${DRIVER_NAME}

#TODO rename?
DRIVER_EXECUTABLE_NAME = nvd

REGISTRY_PRODUCTION ?= nexenta
REGISTRY_DEVELOPMENT ?= 10.3.199.92:5000

VERSION ?= $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
COMMIT ?= $(shell git rev-parse HEAD | cut -c 1-7)
DATETIME ?= $(shell date +'%F_%T')
LDFLAGS ?= \
	-X github.com/Nexenta/nexenta-docker-driver/pkg/driver.Version=${VERSION} \
	-X github.com/Nexenta/nexenta-docker-driver/pkg/driver.Commit=${COMMIT} \
	-X github.com/Nexenta/nexenta-docker-driver/pkg/driver.DateTime=${DATETIME}

.PHONY: all
all: build-development

.PHONY: build-go
build-go:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/${DRIVER_EXECUTABLE_NAME} -ldflags "${LDFLAGS}" ./cmd

.PHONY: plugin-rootfs
ROOTFS_CONTAINER_ID = ""
build-rootfs: clean
	mkdir -p ./plugin/rootfs
	cp config.json ./plugin/
	docker build -f Dockerfile.rootfs -t ${IMAGE_NAME}_${VERSION}:rootfs .
	export ROOTFS_CONTAINER_ID=$(shell docker create ${IMAGE_NAME}_${VERSION}:rootfs); \
	docker export $${ROOTFS_CONTAINER_ID} | tar -x -C ./plugin/rootfs; \
	docker rm -vf $${ROOTFS_CONTAINER_ID}

.PHONY: build-development
build-development: uninstall-development build-rootfs
	cp config.json ./plugin/
	docker plugin create ${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION} ./plugin
	docker plugin enable ${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION}

.PHONY: build-production
build-production: uninstall-production build-rootfs
	cp config.json ./plugin/
	docker plugin create ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION} ./plugin
	docker plugin enable ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION}

.PHONY: push-development
push-development:
	docker plugin push ${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION}

.PHONY: push-production
push-production:
	docker plugin push ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION}

.PHONY: uninstall-development
uninstall-development:
	docker plugin disable -f ${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION} || true
	docker plugin remove -f ${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION} || true

.PHONY: uninstall-production
uninstall-production:
	docker plugin disable -f ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION} || true
	docker plugin remove -f ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION} || true

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf bin plugin
