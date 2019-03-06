# Package configuration
PROJECT = engine
COMMANDS = cmd/srcd
DOCKERFILES ?= cmd/srcd-server/Dockerfile:cli-daemon
PKG_OS ?= darwin linux windows

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_PATH ?= $(shell pwd)/.ci
CI_VERSION ?= v1

MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --branch $(CI_VERSION) --depth 1 $(CI_REPOSITORY) $(CI_PATH);

-include $(MAKEFILE)

LD_FLAGS_INTEGRATION = -X main.version="integration-testing" -X main.build=$(BUILD) -X main.commit=$(COMMIT)

GOTEST_INTEGRATION_TAGS = integration
GOTEST_INTEGRATION = $(GOTEST) -parallel 1 -count 1 -tags="$(GOTEST_INTEGRATION_TAGS)"

INTEGRATION_TEST_BUILD_PATH = "build-integration"
INTEGRATION_TEST_BIN_PATH = $(INTEGRATION_TEST_BUILD_PATH)/bin

GOBUILD_INTEGRATION = $(GOCMD) build -ldflags "$(LD_FLAGS_INTEGRATION)" -tags "$(GOTEST_INTEGRATION_TAGS)"

clean-integration:
	rm -rf $(INTEGRATION_TEST_BUILD_PATH)
build-integration: BUILD_PATH=$(INTEGRATION_TEST_BUILD_PATH)
build-integration: BIN_PATH=$(INTEGRATION_TEST_BIN_PATH)
build-integration: GOBUILD=$(GOBUILD_INTEGRATION)
build-integration: build

build-integration-daemon:
	docker build --build-arg go_build_tags=integration -t srcd/cli-daemon:integration-testing -f cmd/srcd-server/Dockerfile .

test-integration-no-daemon: clean-integration build-integration
	$(GOTEST_INTEGRATION) github.com/src-d/engine/cmd/srcd/cmd/
test-integration: build-integration-daemon test-integration-no-daemon
