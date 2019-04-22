# NexentaStor Docker Volume Driver makefile

DRIVER_NAME = nexentastor-nfs-plugin
IMAGE_NAME ?= ${DRIVER_NAME}

# must be the same as in `config.Name`
DRIVER_EXECUTABLE_NAME = nvd

DOCKER_FILE_PRE_RELEASE = Dockerfile.pre-release
DOCKER_IMAGE_PRE_RELEASE = nexenta-docker-driver-pre-release
DOCKER_CONTAINER_PRE_RELEASE = ${DOCKER_IMAGE_PRE_RELEASE}-container

REGISTRY_PRODUCTION ?= nexenta
REGISTRY_DEVELOPMENT ?= 10.3.199.92:5000

# use git branch as default version if not set by env variable
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
VERSION ?= ${GIT_BRANCH}
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
	docker build --no-cache -f Dockerfile.rootfs -t ${IMAGE_NAME}_${VERSION}:rootfs --build-arg VERSION=${VERSION} .
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

.PHONY: build-production
build-production: uninstall-production build-rootfs
	cp config.json ./plugin/
	docker plugin create ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION} ./plugin

.PHONY: enable-development
enable-development:
	docker plugin enable ${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION}

.PHONY: enable-production
enable-production:
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

.PHONY: release
release:
	@echo "New tag: '${VERSION}'\n\n \
		To change version set enviroment variable 'VERSION=X.X.X make release'.\n\n \
		Confirm that:\n \
		1. New version will be based on current '${GIT_BRANCH}' git branch\n \
		2. Plugin version '${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION}' will be built\n \
		3. Login to hub.docker.com will be requested\n \
		4. Plugin version '${REGISTRY_PRODUCTION}/${IMAGE_NAME}:${VERSION}' will be pushed to hub.docker.com\n \
		5. CHANGELOG.md file will be updated\n \
		6. Git tag '${VERSION}' will be created and pushed to the repository.\n \
		7. Update for 'latest' tag will be suggested, if needed, hub.docker.com 'latest' tag will be updated too.\n\n \
		Are you sure? [y/N]: "
	@(read ANSWER && case "$$ANSWER" in [yY]) true;; *) false;; esac)
	make generate-changelog
	make build-production
	docker login
	make push-production
	git add CHANGELOG.md
	git commit -m "release ${VERSION}"
	git push
	git tag ${VERSION}
	git push --tags
	make update-latest

.PHONY: generate-changelog
generate-changelog:
	@echo "Release tag: ${VERSION}\n"
	docker build -f ${DOCKER_FILE_PRE_RELEASE} -t ${DOCKER_IMAGE_PRE_RELEASE} --build-arg VERSION=${VERSION} .
	-docker rm -f ${DOCKER_CONTAINER_PRE_RELEASE}
	docker create --name ${DOCKER_CONTAINER_PRE_RELEASE} ${DOCKER_IMAGE_PRE_RELEASE}
	docker cp \
		${DOCKER_CONTAINER_PRE_RELEASE}:/go/src/github.com/Nexenta/nexenta-docker-driver/CHANGELOG.md \
		./CHANGELOG.md
	docker rm ${DOCKER_CONTAINER_PRE_RELEASE}

.PHONY: update-latest
update-latest:
	@echo "\nIs this the latest version of the plugin?\n"
	@echo "If yes, this version will be pushed as 'latest' to hub.docker.com:"
	@./plugin/rootfs/bin/nvd --version
	@echo "\nPublish 'latest' version? [y/N]: "
	@(read ANSWER && case "$$ANSWER" in [yY]) true;; *) false;; esac)
	cp config.json ./plugin/
	docker plugin create ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:latest ./plugin
	docker plugin push ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:latest

.PHONY: clean
clean:
	-go clean -r -x
	-rm -rf ./bin
	-rm -rf ./plugin
	-docker rmi -f ${IMAGE_NAME}_${VERSION}:rootfs
