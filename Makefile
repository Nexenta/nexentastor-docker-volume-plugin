# NexentaStor Docker Volume Driver makefile

DRIVER_NAME = nexentastor-nfs-plugin
IMAGE_NAME ?= ${DRIVER_NAME}

# must be the same as in `config.Name`
DRIVER_EXECUTABLE_NAME = nvd

REGISTRY_PRODUCTION ?= nexenta
REGISTRY_DEVELOPMENT ?= 10.3.199.92:5000

VERSION ?= $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
COMMIT ?= $(shell git rev-parse HEAD | cut -c 1-7)
DATETIME ?= $(shell date -u +'%F_%T')
LDFLAGS ?= \
	-X github.com/Nexenta/nexenta-docker-driver/pkg/config.Version=${VERSION} \
	-X github.com/Nexenta/nexenta-docker-driver/pkg/config.Commit=${COMMIT} \
	-X github.com/Nexenta/nexenta-docker-driver/pkg/config.DateTime=${DATETIME}

.PHONY: all
all: build-development

.PHONY: build-go
build-go:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/${DRIVER_EXECUTABLE_NAME} -ldflags "${LDFLAGS}" ./cmd

.PHONY: plugin-rootfs
build-rootfs: clean
	mkdir -p ./plugin/rootfs
	docker build --no-cache -f Dockerfile.rootfs -t ${IMAGE_NAME}_${VERSION}:rootfs .
	docker create ${IMAGE_NAME}_${VERSION}:rootfs > /tmp/.nvdContainerId
	@echo "Temporary container ID:"
	@cat /tmp/.nvdContainerId
	docker export $$(cat /tmp/.nvdContainerId) | tar -x -C ./plugin/rootfs
	docker rm $$(cat /tmp/.nvdContainerId)
	rm /tmp/.nvdContainerId
	@echo "---------------------------------------"
	@echo "Plugin version:"
	@./plugin/rootfs/bin/${DRIVER_EXECUTABLE_NAME} --version
	@echo "Current UTC time: ($$(date -u +'%F_%T'))"
	@echo "---------------------------------------"

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
	-rm -rf ./bin
	-rm -rf ./plugin
	-docker rmi -f ${IMAGE_NAME}_${VERSION}:rootfs
