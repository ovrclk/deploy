#!/bin/bash

#
# Set up a kubernetes environment with kind.
#
# * Install Akash CRD
# * Optionally install metrics-server

# TODO: ensure this is run after the chain script or have optional akash repo pull here too?

# root dir is the akash directory
akashdir="$GOPATH/src/github.com/ovrclk/akash"

install_crd() {
  echo "Creating akash custom resource definition..."
  kubectl apply -f "$akashdir/pkg/apis/akash.network/v1/crd.yaml" > /dev/null
}

install_metrics() {
  # https://github.com/kubernetes-sigs/kind/issues/398#issuecomment-621143252
  echo "Creating kube metrics server..."
  kubectl apply -f "$akashdir/script/kind-metrics-server.yaml" > /dev/null

  count=1
  continue=true
  while $continue; do
    kubectl top nodes > /dev/null 2>&1
    if [ ! $? -eq 0 ]; then
      count=$(($count+1))
      if ! (( $count % 10 )); then
        echo "[$count] waiting for metrics..."
      fi
      sleep 1
    else
      continue=false
    fi
  done

  echo "Waiting for kube ingress..."
  kubectl wait pod --namespace ingress-nginx \
    --for=condition=Ready \
    --selector=app.kubernetes.io/component=controller \
    --timeout=120s > /dev/null

  echo "Cluster available!"
}

usage() {
  cat <<EOF
  Install k8s dependencies for integration tests against "KinD"

  Usage: $0 [crd|metrics]

  crd:     install the akash CRDs
  metrics: install CRDs, metrics-server and wait for metrics to be available
EOF
  exit 1
}

case "${1:-metrics}" in
  crd)
    install_crd
    ;;
  metrics)
    install_crd
    install_metrics
    ;;
  *) usage;;
esac
