SHELL := /bin/bash

# Include custom variables to modify behavior.
GITROOT ?= $(shell git rev-parse --show-toplevel 2>/dev/null || echo $(CURDIR))
ifneq ("$(wildcard $(GITROOT)/Makefile.vars)","")
	include Makefile.vars
endif

# Setup default go compiler environment.
export GO ?= go
export GOPATH ?= $(shell $(GO) env GOPATH)
export GOBIN ?= $(GOPATH)/bin
# Setup go-make to utilize desired build and config scripts.
GOMAKE_DEP ?= github.com/tkrop/go-make@v0.0.74
# Request targets from go-make targets target.
TARGETS := $(shell command -v $(GOBIN)/go-make >/dev/null || \
	$(GO) install $(GOMAKE_DEP) >/dev/stderr && \
	$(GOBIN)/go-make show-targets 2>/dev/null)
# Declare all targets phony to make them available for auto-completion.
.PHONY:: $(TARGETS)

# Delegate all targets to go-make in a single call stubbing other targets.
$(eval $(wordlist 1,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))::;@:)
$(firstword $(MAKECMDGOALS) all)::
	@$(GOBIN)/go-make $(MAKECMDGOALS);
