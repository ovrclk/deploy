#!/bin/bash

killall akashctl &> /dev/null

AKASH_CLIENT="./data/client/"

echo "Running provider daemon..."
AKASH_DEPLOYMENT_INGRESS_STATIC_HOSTS="false"  akashctl --home $AKASH_CLIENT provider run \
    --from "provider" --cluster-k8s \
    --gateway-listen-address localhost:8080 > ./data/provider.log 2>&1 &