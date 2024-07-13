# BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
# COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# Update the ldflags with the app, client & server names
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=ludiumappd \
	-X github.com/cosmos/cosmos-sdk/version.AppName=ludiumappd \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

###########
# Install #
###########

all: install

install:
	@echo "--> ensure dependencies have not been modified"
	@go mod verify
	@echo "--> installing ludiumappd"
	@go install $(BUILD_FLAGS) -mod=readonly ./ludiumappd

init-demo-chain: install
	@echo "--> init new demo chain"
	./scripts/init.sh 

	@echo "--> start the chain"
	./scripts/start.sh
	