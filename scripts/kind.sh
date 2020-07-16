#!/bin/bash

cluster="kube"
config="./scripts/kind-config-80.yaml"
ingress="https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/kind/deploy.yaml"

delete_cluster() {
  echo "Deleting kind cluster..."
  kind delete cluster -q --name $cluster
}

start_cluster() {
  echo "Starting kind cluster..."
  kind create cluster -q --config $config --name $cluster
}

setup() {
  echo "Creating kube ingress..."
  kubectl apply -f $ingress > /dev/null
  ./scripts/setup-kind.sh
}

usage() {
  cat <<EOF
  Start kind clusters

  Usage: $0 [delete|create]

  create: delete existing kind cluster and create a new one
  delete: delete kind cluster
EOF
  exit 1
}

case "${1:-create}" in
  create)
    delete_cluster
    start_cluster
    setup
    ;;
  delete)
    delete_cluster
    ;;
  *) usage;;
esac
