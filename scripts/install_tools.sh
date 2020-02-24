#!/bin/sh

# set -x

DIR="`dirname \"$0\"`"
ROOTDIR="`( cd \"$DIR/../\" && pwd )`"

PROTOC_VERSION=3.11.4

KUBECTL_VERSION=1.15.6
KIND_VERSION=0.7.0
KUSTOMIZE_VERSION=3.5.4
MIGRATE_VERSION=4.9.1
NODE_VERSION=13.9.0

arch=`uname -m`
os=`uname -s`
protoc_os=$os

case "${os}" in
  Darwin*)
        os=darwin
        protoc_os=osx
        ;; 
  Linux*) 
        os=linux
        protoc_os=linux
        ;;
  *)  
        echo "unsupported: $OSTYPE" 
        exit 1
        ;;
esac

__install_protoc() {
    local asset=protoc-${PROTOC_VERSION}-${protoc_os}-${arch}.zip
    local protoc_url=https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${asset}
    echo "Download  $protoc_url"
    
    curl -sLJO $protoc_url
    unzip -d ${ROOTDIR}/.tools ${asset}
    rm -rf ${asset}
}

__install_kubectl() {

    local kubectl_url=https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${os}/amd64/kubectl
    echo "Download  $kubectl_url"

    curl -sLJO ${kubectl_url}
    chmod +x kubectl
    mv kubectl ${ROOTDIR}/.tools/bin
}

__install_kind() {
    local kind_url=https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-${os}-amd64
    echo "Download  $kind_url"

    curl -sLJO ${kind_url}
    chmod +x ./kind-${os}-amd64
    mv ./kind-${os}-amd64 ${ROOTDIR}/.tools/bin/kind
}


__install_kustomize() {
    local kustomize_url=https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_${os}_amd64
    echo "Download  $kustomize_url"

    curl -sLJO ${kustomize_url}
    chmod +x ./kustomize_${KUSTOMIZE_VERSION}_${os}_amd64
    mv ./kustomize_${KUSTOMIZE_VERSION}_${os}_amd64 ${ROOTDIR}/.tools/bin/kustomize
}

__install_migrate() {
    local asset=migrate.${os}-amd64.tar.gz
    local migrate_url=https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/${asset}
    echo "Download  $migrate_url"

    curl -fsLJO $migrate_url
    tar -C ${ROOTDIR}/.tools/bin -zxf ${asset}
    mv ${ROOTDIR}/.tools/bin/migrate.${os}-amd64 ${ROOTDIR}/.tools/bin/migrate
    rm -rf ${asset}
}

__install_gotools() {
    go install -mod=vendor golang.org/x/tools/cmd/goimports
    go install -mod=vendor github.com/golangci/golangci-lint/cmd/golangci-lint
    go install -mod=vendor github.com/cnative/servicebuilder
    go install -mod=vendor github.com/golang/protobuf/protoc-gen-go
    go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
    go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
    go install -mod=vendor github.com/golang/mock/mockgen
    go install -mod=vendor github.com/cloudflare/cfssl/cmd/cfssl
    go install -mod=vendor github.com/cloudflare/cfssl/cmd/cfssljson
    go install -mod=vendor github.com/go-bindata/go-bindata/go-bindata
}

__install_node() {
    
    local asset=node-v${NODE_VERSION}-${os}-x64.tar.gz
    local node_url=https://nodejs.org/dist/v${NODE_VERSION}/${asset}
    echo "Download  $node_url"
    curl -fsLJO $node_url
    tar -C ${ROOTDIR}/.tools -zxf ${asset}
    mv ${ROOTDIR}/.tools/node-v${NODE_VERSION}-${os}-x64 ${ROOTDIR}/.tools/node
    rm -rf ${asset}
}

rm -rf ${ROOTDIR}/.tools
mkdir -p ${ROOTDIR}/.tools/bin

__install_node

__install_protoc

__install_kubectl

__install_kustomize

__install_kind

__install_gotools

__install_migrate

