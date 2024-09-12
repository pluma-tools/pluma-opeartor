ifeq ($(shell uname),Darwin)
SEDI=sed -i ""
else
SEDI=sed -i
endif


.PHONY: gen-proto
gen-proto: format-proto
	./gen.sh

gen-test:
	go test .
.PHONY: gen-test

.PHONY: clean-proto
clean-proto:
	./clean.sh

fix-import:
	for f in $$(find . -name "*.pb.go"); do \
      if grep github.com/golang/protobuf/ptypes/struct $$f &>/dev/null; then \
          $(SEDI) 's#github.com/golang/protobuf/ptypes/struct#google.golang.org/protobuf/types/known/structpb#g' $$f; \
      fi; \
    done

manifests: controller-gen fix-import ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	out=$$($(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role crd:ignoreUnexportedFields=true \
		paths="./..." output:crd:artifacts:config=config/crd/bases &>/dev/stdout); \
		if [[ $$? != 0 ]]; then c=$$(echo "$$out" | wc -l | xargs); echo "errorline: $$c"; if [[ $$c != 3 ]]; then echo "$$out"; exit 1; fi; fi
	for f in config/crd/bases/*; do \
		yq eval '(.. | select(has("enum") and .type == "integer") | .type) |= "string"' $$f -i ; \
	done

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0)

CODE_GENERATOR = $(shell echo bash $${GOPATH}/pkg/mod/k8s.io/code-generator@v0.25.1/generate-groups.sh)

.PHONY: gen-client
gen-client:
	@echo "Generating client code..."
	d=$$(mktemp -d) && $(CODE_GENERATOR) \
		all \
		pluma.io/api/client \
		pluma.io/api \
		"operator:v1alpha1" \
		--output-base $${d} \
		--go-header-file hack/boilerplate.go.txt && \
		rm -rf client && \
		mv $${d}/pluma.io/api/client .
	find client | grep -E "/typed/.*/fake/fake_.*.go" | xargs -I{} $(SEDI) 's/Group: "operator"/Group: "operator.pluma.io"/g' {}
	rm -rf "$${d}"
	@if [ ! -d "client" ]; then \
		echo "Error: Client code was not generated. Please check the following:"; \
		echo "1. Ensure CODE_GENERATOR is correctly set and points to a valid generate-groups.sh script."; \
		echo "2. Verify that the input paths and package names are correct."; \
		echo "3. Check for any error messages from the code generator."; \
		echo "4. Make sure you have the necessary dependencies installed."; \
		exit 1; \
	else \
		echo "Client code generation completed successfully."; \
	fi

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -ex ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
mkdir -p $(PROJECT_DIR)/bin; \
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

format-proto:
	find . -name "*.proto" -exec clang-format -style=file -i {} \;

lint-proto:
	go test --race -v ./...
.PHONY: lint-proto


