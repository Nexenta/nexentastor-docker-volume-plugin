# NexentaStor Docker Volume plugin makefile
#
# Test options to be set before run tests:
# - NOCOLOR=1                   # disable colors
# - TEST_DOCKER_IP=10.3.199.249 # for e2e Docker tests
#

PLUGIN_NAME = nexentastor-docker-volume-plugin
IMAGE_NAME ?= ${PLUGIN_NAME}

GIT_REPOSITORY = github.com/Nexenta/${PLUGIN_NAME}

DOCKER_FILE_TESTS = Dockerfile.tests
DOCKER_IMAGE_TESTS = ${PLUGIN_NAME}-tests
DOCKER_FILE_CHANGELOG = Dockerfile.changelog
DOCKER_IMAGE_CHANGELOG = ${PLUGIN_NAME}-changelog
DOCKER_CONTAINER_CHANGELOG = ${DOCKER_IMAGE_CHANGELOG}-container

REGISTRY_PRODUCTION ?= nexenta
REGISTRY_DEVELOPMENT ?= 10.3.199.92:5000

# use git branch as default version if not set by env variable
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
VERSION ?= ${GIT_BRANCH}
COMMIT ?= $(shell git rev-parse HEAD | cut -c 1-7)
DATETIME ?= $(shell date -u +'%F_%T')
LDFLAGS ?= \
	-X ${GIT_REPOSITORY}/pkg/config.Version=${VERSION} \
	-X ${GIT_REPOSITORY}/pkg/config.Commit=${COMMIT} \
	-X ${GIT_REPOSITORY}/pkg/config.DateTime=${DATETIME}

.PHONY: all
all: build-development

.PHONY: build-go
build-go:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/${PLUGIN_NAME} -ldflags "${LDFLAGS}" ./cmd

.PHONY: plugin-rootfs
build-rootfs: clean
	mkdir -p ./plugin/rootfs
	docker build --no-cache -f Dockerfile.rootfs -t ${IMAGE_NAME}_${VERSION}:rootfs --build-arg VERSION=${VERSION} .
	docker create ${IMAGE_NAME}_${VERSION}:rootfs > /tmp/.nexentastorDockerVolumePluginContainerId
	@echo "Temporary container ID:"
	@cat /tmp/.nexentastorDockerVolumePluginContainerId
	docker export $$(cat /tmp/.nexentastorDockerVolumePluginContainerId) | tar -x -C ./plugin/rootfs
	docker rm $$(cat /tmp/.nexentastorDockerVolumePluginContainerId)
	rm /tmp/.nexentastorDockerVolumePluginContainerId
	@echo "---------------------------------------"
	@echo "Plugin version:"
	@./plugin/rootfs/bin/${PLUGIN_NAME} --version
	@echo "Current UTC time:                               ($$(date -u +'%F_%T'))"
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


.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit:
	go test ./tests/unit/arrays -v -count 1
	go test ./tests/unit/config -v -count 1
.PHONY: test-unit-container
test-unit-container:
	docker build -f ${DOCKER_FILE_TESTS} -t ${IMAGE_NAME}-test --build-arg VERSION=${VERSION} .
	docker run -i --rm -e NOCOLORS=${NOCOLORS} ${IMAGE_NAME}-test test-unit

# run e2e docker tests using image from local docker registry
.PHONY: test-e2e-docker-development
test-e2e-docker-development: check-env-TEST_DOCKER_IP
	go test tests/e2e/plugin_test.go -v -count 1 -failfast \
		--ssh="root@${TEST_DOCKER_IP}" \
		--plugin="${REGISTRY_DEVELOPMENT}/${IMAGE_NAME}:${VERSION}" \
		--config="./_configs/single-ns.yaml"
.PHONY: test-e2e-docker-development-container
test-e2e-docker-development-container: check-env-TEST_DOCKER_IP
	docker build -f ${DOCKER_FILE_TESTS} -t ${DOCKER_IMAGE_TESTS} .
	docker run -i --rm -v ${HOME}/.ssh:/root/.ssh:ro \
		-e VERSION=${VERSION} -e NOCOLORS=${NOCOLORS} -e TEST_DOCKER_IP=${TEST_DOCKER_IP} \
		${DOCKER_IMAGE_TESTS} test-e2e-docker-development

.PHONY: check-env-TEST_DOCKER_IP
check-env-TEST_DOCKER_IP:
ifeq ($(strip ${TEST_DOCKER_IP}),)
	$(error "Error: environment variable TEST_DOCKER_IP is not set (e.i. 10.3.199.249)")
endif

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
	docker build -f ${DOCKER_FILE_CHANGELOG} -t ${DOCKER_IMAGE_CHANGELOG} --build-arg VERSION=${VERSION} .
	-docker rm -f ${DOCKER_CONTAINER_CHANGELOG}
	docker create --name ${DOCKER_CONTAINER_CHANGELOG} ${DOCKER_IMAGE_CHANGELOG}
	docker cp \
		${DOCKER_CONTAINER_CHANGELOG}:/go/src/${GIT_REPOSITORY}/CHANGELOG.md \
		./CHANGELOG.md
	docker rm ${DOCKER_CONTAINER_CHANGELOG}

.PHONY: update-latest
update-latest:
	@echo "\nIs this the latest version of the plugin?\n"
	@echo "If yes, this version will be pushed as 'latest' to hub.docker.com:"
	@./plugin/rootfs/bin/${PLUGIN_NAME} --version
	@echo "\nPublish 'latest' version? [y/N]: "
	@(read ANSWER && case "$$ANSWER" in [yY]) true;; *) false;; esac)
	cp config.json ./plugin/
	docker plugin disable -f ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:latest || true
	docker plugin remove -f ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:latest || true
	docker plugin create ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:latest ./plugin
	docker plugin push ${REGISTRY_PRODUCTION}/${IMAGE_NAME}:latest

.PHONY: clean
clean:
	-go clean -r -x
	-rm -rf ./bin
	-rm -rf ./plugin
	-docker rmi -f ${IMAGE_NAME}_${VERSION}:rootfs
