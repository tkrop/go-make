SHELL := /bin/bash

# Include custom variables to modify behavior.
ifneq ("$(wildcard Makefile.vars)","")
	include Makefile.vars
else
	$(warning warning: please customize variables in Makefile.vars)
endif

#GOBIN := $(CURDIR)/build/bin
#PATH := $(GOBIN):$(PATH)
GOBIN ?= $(shell go env GOPATH)/bin
GOMAKE ?= github.com/tkrop/go-make@v0.0.27
TARGETS := $(shell command -v $(GOBIN)/go-make >/dev/null || \
	make -f config/Makefile.base install >/dev/stderr &&  \
	$(GOBIN)/go-make targets)

# Declare all targets phony to make them available for auto-completion.
.PHONY:: $(TARGETS)

# Delegate all targets to go-make in a single call suppressing other targets.
$(eval $(wordlist 1,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))::;@:)
$(firstword $(MAKECMDGOALS) all)::
	@$(GOBIN)/go-make $(strip $(MAKEFLAGS) $(MAKECMDGOALS));
