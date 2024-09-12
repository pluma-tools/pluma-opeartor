SHELL := /bin/bash

# allow optional per-repo overrides
-include Makefile.overrides.mk

# If we are not in build container, we need a workaround to get environment properly set
# Write to file, then include
$(shell mkdir -p out)
$(shell $(shell pwd)/scripts/setup_env.sh envfile > out/.env)
include out/.env
# An export free of arguments in a Makefile places all variables in the Makefile into the
# environment. This behavior may be surprising to many that use shell often, which simply
# displays the existing environment
export

export GOBIN ?= $(GOPATH)/bin
include Makefile.core.mk
