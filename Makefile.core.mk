GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOPROXY ?= http://10.6.100.13:8081/repository/go-proxy/
GOSUMDB ?= sum.golang.org http://10.6.100.13:8081/repository/gosum/
GOPRIVATE ?= gitlab.daocloud.cn

BUILD_ARCH ?= linux/$(GOARCH)
OFFLINE_ARCH ?= amd64

HUB ?= release-ci.daocloud.io/mspider
HUB_CI = release-ci.daocloud.io/mspider
HELM_REPO ?= https://release-ci.daocloud.io/chartrepo/mspider
PROD_NAME ?= mspider
MINOR_VERSION ?= v0.30
VERSION ?= $(MINOR_VERSION)-dev-$(shell git rev-parse --short=8 HEAD)

REGISTRY_USER_NAME?=
REGISTRY_PASSWORD?=

PUSH_IMAGES ?= 1

RETRY_LIMIT := 3

NPM_TOKEN ?=

OFFLINE ?= 0

CI_IMAGE_VER ?= $(UNIFIED_CI_IMAGE_VER)


gen-proto:
	make -C apis gen-proto
clean-proto:
	make -C apis clean-proto
generate:
	make -C apis generate

ctl-manifests:
	make -C apis manifests
	bash ./scripts/copy-crds.sh apis/config/crd/bases/operator.pluma.io_helmapps.yaml manifests/pluma/templates

format-shell:
	shfmt -i 4 -l -w ./scripts
format-go:
	goimports -local gitlab.daocloud.cn/nicole.li/pluma-opeartor -w .
	gofmt -w .


format: format-go format-shell 

gen: clean-proto gen-proto generate ctl-manifests gen-client format

gen-client:
	make -C apis gen-client

gen-istio-values:
	./scripts/gen-istio-values.sh
