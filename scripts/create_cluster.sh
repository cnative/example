#!/bin/bash
CLUSTER_NAME=${CLUSTER_NAME:-cnative-local}
echo $CLUSTER_NAME

DIR="`dirname \"$0\"`"
ROOTDIR="`( cd \"$DIR/../\" && pwd )`"  # normalized project root dir
kubectl=${ROOTDIR}/.tools/bin/kubectl

export KUBECONFIG=$(${ROOTDIR}/.tools/bin/kind get kubeconfig-path --name ${CLUSTER_NAME})

if test -f "$KUBECONFIG"; then
    echo "cluster already exists. use kubectl --kubeconfig=$KUBECONFIG to use the cluster"
else
    ${ROOTDIR}/.tools/bin/kind create cluster --name ${CLUSTER_NAME}
    n=0; until ((n >= 60)); do kubectl -n default get serviceaccount default -o name && break; n=$((n + 1)); sleep 1; done; ((n < 60))

    if [ "$CLUSTER_NAME" == "cnative-integ" ]; then
        # install cert-manager
        _cert_manager_ns=$(mktemp -d)/cert-manager-ns.yaml
        cat > ${_cert_manager_ns} <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: cert-manager
  labels:
    certmanager.k8s.io/disable-validation: "true"
EOF
        $kubectl create -f ${_cert_manager_ns}
        $kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.8.0/cert-manager.yaml
        $kubectl wait --for=condition=available --timeout=600s deployment/cert-manager deployment/cert-manager-webhook --namespace cert-manager
    fi
fi