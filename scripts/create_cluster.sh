#!/bin/bash

DIR="`dirname \"$0\"`"
ROOTDIR="`( cd \"$DIR/../\" && pwd )`"  # normalized project root dir
CERTMGR="https://github.com/jetstack/cert-manager/releases/download/v0.13.1/cert-manager.yaml"
kubectl=${ROOTDIR}/.tools/bin/kubectl

export KUBECONFIG=${ROOTDIR}/.tools/kubeconfig

if ! ${ROOTDIR}/.tools/bin/kind create cluster --name ${CLUSTER_NAME} --wait 3m; then
    exit 1;
fi

# install cert-manager
$kubectl create namespace cert-manager
$kubectl apply --validate=false -f ${CERTMGR}

# install postgres-operator
$kubectl apply -k ./deployments/postgres-operator