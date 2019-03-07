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

LD_FLAGS_IMAGE_PREFIX = -X github.com/src-d/engine/components.imageNamePrefix=integration-testing-scrd-cli-
LD_FLAGS_INTEGRATION = -X main.version=integration-testing -X main.build=$(BUILD) -X main.commit=$(COMMIT) $(LD_FLAGS_IMAGE_PREFIX)

GOTEST_INTEGRATION = $(GOTEST) -parallel 1 -count 1 -tags=integration -ldflags "$(LD_FLAGS_INTEGRATION)"

INTEGRATION_TEST_BUILD_PATH = "build-integration"
INTEGRATION_TEST_BIN_PATH = $(INTEGRATION_TEST_BUILD_PATH)/bin

GOBUILD_INTEGRATION = $(GOCMD) build -ldflags "$(LD_FLAGS_INTEGRATION)"

clean-integration:
	rm -rf $(INTEGRATION_TEST_BUILD_PATH)
build-integration: BUILD_PATH=$(INTEGRATION_TEST_BUILD_PATH)
build-integration: BIN_PATH=$(INTEGRATION_TEST_BIN_PATH)
build-integration: GOBUILD=$(GOBUILD_INTEGRATION)
build-integration: build

build-integration-daemon:
	docker build --build-arg go_ldflags="$(LD_FLAGS_IMAGE_PREFIX)" -t srcd/cli-daemon:integration-testing -f cmd/srcd-server/Dockerfile .

test-integration-no-daemon: clean-integration build-integration
	$(GOTEST_INTEGRATION) github.com/src-d/engine/cmd/srcd/cmd/
test-integration: build-integration-daemon test-integration-no-daemon
