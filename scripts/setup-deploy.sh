#!/bin/bash

DEPLOY_DATA="$HOME/.akash-deploy"
CHAIN_ID="testchain"
RPC_ADDR="http://localhost:26657"
AKASH_DATA="$(pwd)/data"
CLIENT_DATA="$AKASH_DATA/client"
KEYPASS="12345678"
KEYFILE="key.priv"
CONFIG="$DEPLOY_DATA/config.yaml"


# TODO: ensure that akash is installed and the repo exists locally

# Ensure user understands what will be deleted
if [[ -d $DEPLOY_DATA ]] && [[ ! "$1" == "skip" ]]; then
  read -p "$0 will delete $DEPLOY_DATA folder. Do you wish to continue? (y/n): " -n 1 -r
  echo 
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
  fi
fi

# Delete the old config directory
echo "Removing old ~/.akash-deploy and reconfiguring..."
rm -rf $DEPLOY_DATA &> /dev/null
mkdir -p $DEPLOY_DATA &> /dev/null

# Export the deployment private key
printf "$KEYPASS\n$KEYPASS\n" | akashctl --home $CLIENT_DATA keys export main 2> $DEPLOY_DATA/$KEYFILE

# Create the configuration file
echo "chain-id: $CHAIN_ID" > $CONFIG
echo "rpc-addr: $RPC_ADDR" >> $CONFIG
echo "keyfile: $KEYFILE" >> $CONFIG
echo "keypass: $KEYPASS" >> $CONFIG
