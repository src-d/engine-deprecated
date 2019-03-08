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

GOTEST_INTEGRATION = $(GOTEST) -parallel 1 -count 1 -tags=integration -ldflags "$(LD_FLAGS)"

test-integration-no-build:
	$(GOTEST_INTEGRATION) github.com/src-d/engine/cmd/srcd/cmd/

test-integration: clean build docker-build test-integration-no-build
