# Package configuration
PROJECT = engine
COMMANDS = cmd/srcd
DOCKERFILES ?= cmd/srcd-server/Dockerfile:cli-daemon
PKG_OS ?= darwin linux windows

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_PATH ?= $(shell pwd)/.ci
CI_VERSION ?= v1

TEST_PRUNE_WITH_IMAGE ?= false

MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --branch $(CI_VERSION) --depth 1 $(CI_REPOSITORY) $(CI_PATH);

-include $(MAKEFILE)

GOTEST_INTEGRATION = $(GOTEST) -timeout 20m -parallel 1 -count 1 -tags=integration -ldflags "$(LD_FLAGS)"

OS := $(shell uname)

ifeq ($(OS),Darwin)
test-integration-clean:
	$(eval TMPDIR_TEST := $(PWD)/integration-test-tmp)
	$(eval GOTEST_INTEGRATION := TMPDIR=$(TMPDIR_TEST) $(GOTEST_INTEGRATION))
	rm -rf $(TMPDIR_TEST)
	mkdir $(TMPDIR_TEST)
else
test-integration-clean:
endif

test-integration-no-build: test-integration-clean
	TEST_PRUNE_WITH_IMAGE=false $(GOTEST_INTEGRATION) github.com/src-d/engine/cmdtests/
	$(GOTEST_INTEGRATION) github.com/src-d/engine/cmdtests/ -run TestPruneTestSuite/TestRunningContainersWithImages

test-integration: clean build docker-build test-integration-no-build
