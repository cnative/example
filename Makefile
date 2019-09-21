export VERSION    ?= dev
export GIT_COMMIT ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo unknown)
export LD_FLAGS = -X "main.gitCommit=$(GIT_COMMIT)" -X "main.version=$(VERSION)"

export GOBIN = $(abspath .)/.tools/bin
export PATH := $(GOBIN):$(abspath .)/bin:$(PATH)
export GO111MODULE=on
export CGO_ENABLED=0

export V = 0
export Q = $(if $(filter 1,$V),,@)
export M = $(shell printf "\033[34;1m▶\033[0m")

help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

deps:
	@go get -d -v ./...

vendor: deps ; $(info $(M) vendoring …)
	@go mod tidy && go mod vendor && cp -r $(shell go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway)/third_party vendor/github.com/grpc-ecosystem/grpc-gateway/ && \
		chmod -R 755 vendor/github.com/grpc-ecosystem/grpc-gateway/third_party

install-deptools: ## install dependent go tools
	$Q ./scripts/install_tools_check.sh \
	; $(info $(M) installing protoc, golint, protoc-gen-go, protoc-gen-grpc-gateway, protoc-gen-swagger …)

gen:
	$Q go generate -mod=vendor ./pkg/api ./pkg/state ./db/postgres; $(info $(M) generating grpc api server handler, gateway, swagger and metrics & trace store …)

fmt: ; $(info $(M) formatting …) @ ## run go fmt on all source files
	@go fmt $(shell go list ./... | grep -v db/postgres/migrations) && goimports -w -local github.com/cnative/example ./cmd ./pkg

vet: ; $(info $(M) vetting …) @ ## run go vet on all source files
	@go vet -mod=vendor ./...

lint: ; $(info $(M) linting …) @ ## run golint
	@./.tools/bin/golangci-lint --skip-dirs ./vendor run ./...

build-reports-server:
	$Q go build -mod=vendor -ldflags '$(LD_FLAGS)' -o bin/reports-server ./cmd/reports-server ; $(info $(M) building reports-server executable …)

build-linux:
	$Q mkdir -p ./bin/linux_amd64 && \
		GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags '$(LD_FLAGS)' -o bin/linux_amd64/reports-server ./cmd/reports-server; \
			$(info $(M) building executables for linux …)

build: install-deptools gen fmt vet lint build-reports-server build-linux build-testbin-linux; $(info $(M) done. ) @ ## build service

test: ; $(info $(M) testing …) @ ## run go tests with race detector
	$Q CGO_ENABLED=1 go test -mod=vendor $(GO_TEST_FLAGS) $(shell go list ./... | grep -v /tests/e2e)

benchmark: ; $(info $(M) benchmark …) @ ## run go benchmark
	$Q CGO_ENABLED=1 go test -benchmem -bench . $(shell go list ./... | grep -v /tests/e2e)

cluster-integ:
	$Q CLUSTER_NAME=cnative-integ ./scripts/create_cluster.sh; $(info $(M) creating cnative-integ cluster …)

build-testbin-linux:
	$Q mkdir -p ./bin/linux_amd64 && GOOS=linux GOARCH=amd64 go test -c -mod=vendor ./tests/e2e -o ./bin/linux_amd64/cnative-e2e-tests; $(info $(M) building e2e test binary for linux …)

e2e-tests:
	$Q ./scripts/e2e_tests.sh; $(info $(M) running end to end tests `…)

cluster-integ-delete:
	$Q kind delete cluster --name cnative-integ ; $(info $(M) deleting cnative-integ cluster …)

cluster-local: local-certs ## start a local kubernetes cluster
	$Q ./scripts/create_cluster.sh && \
		KUBECONFIG=$(shell ${GOBIN}/kind get kubeconfig-path --name="cnative-local") kubectl apply -k deployments/localhost \
	; $(info $(M) creating cnative-local cluster `…)

cluster-local-delete: ## delete local kubernetes cluster
	$Q kind delete cluster --name cnative-local ; $(info $(M) deleting cnative-local cluster …)

clean: ; $(info $(M) cleaning …)	@ ## cleanup everything
	@rm -rf bin web/build
	@rm -rf test/tests.* test/coverage.*

local-certs: ; $(info $(M) generating local certs …)	@ ## generate TLS certs for local development
	$Q ./scripts/gen_tls_certs.sh > /dev/null 2>&1

gen-web.go:
	$Q ./scripts/gen-web.go.sh; $(info $(M) generating web.go..)

install-ui-deps:
	$Q cd web && npm install; $(info $(M) installing node_modules dependencies...)

build-ui: install-ui-deps ; $(info $(M) building ui... ) @ ## build ui service
	$Q cd web && npm run-script build ; $(info $(M) running npm build …)


