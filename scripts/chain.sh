#!/bin/bash

AKASH_REPO="$GOPATH/src/github.com/ovrclk/akash"
AKASH_BRANCH=v0.7.7
AKASH_DATA="$(pwd)/data"
NODE_DATA="$AKASH_DATA/node"
CLIENT_DATA="$AKASH_DATA/client"
CHAIN_ID="testchain"

# TODO: ensure that akash is installed and the repo exists locally

# Ensure user understands what will be deleted
if [[ -d $AKASH_DATA ]] && [[ ! "$1" == "skip" ]]; then
  read -p "$0 will delete \$(pwd)/data folder. Do you wish to continue? (y/n): " -n 1 -r
  echo 
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
  fi
fi

# kill any old akashd processes
killall akashd &> /dev/null

# delete the old chain
rm -rf $AKASH_DATA &> /dev/null

set -e

echo "Generating akash configurations..."
mkdir -p $NODE_DATA $CLIENT_DATA && cd $AKASH_DATA
akashd init --chain-id $CHAIN_ID $CHAIN_ID --home $NODE_DATA &> /dev/null

cfgpth="$AKASH_DATA/node/config/config.toml"
if [ "$(uname)" = "Linux" ]; then
  # TODO: Just index *some* specified tags, not all
  sed -i 's/index_all_keys = false/index_all_keys = true/g' $cfgpth
  
  # Set proper database backend default
  sed -i 's/"leveldb"/"goleveldb"/g' $cfgpth
  
  # Make blocks run faster than normal
  sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $cfgpth
  sed -i 's/timeout_propose = "3s"/timeout_propose = "1s"/g' $cfgpth
else
  # TODO: Just index *some* specified tags, not all
  sed -i '' 's/index_all_keys = false/index_all_keys = true/g' $cfgpth

  # Set proper database backend default
  sed -i '' 's/"leveldb"/"goleveldb"/g' $cfgpth

  # Make blocks run faster than normal
  sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $cfgpth
  sed -i '' 's/timeout_propose = "3s"/timeout_propose = "1s"/g' $cfgpth
fi

# configure cli
akashctl config --home $CLIENT_DATA chain-id $CHAIN_ID &> /dev/null
akashctl config --home $CLIENT_DATA output json &> /dev/null
akashctl config --home $CLIENT_DATA keyring-backend test &> /dev/null
akashctl config --home $CLIENT_DATA indent true &> /dev/null
akashctl config --home $CLIENT_DATA node http://localhost:26657 &> /dev/null

# add keys for transactions
akashctl --home $CLIENT_DATA keys add main &> /dev/null
akashctl --home $CLIENT_DATA keys add provider &> /dev/null
akashctl --home $CLIENT_DATA keys add validator &> /dev/null
akashctl --home $CLIENT_DATA keys add other &> /dev/null

# ensure denom in genesis is `akash`
cp "$NODE_DATA/config/genesis.json" "$NODE_DATA/config/genesis.json.orig"
cat "$NODE_DATA/config/genesis.json.orig" | \
    jq -rM '(..|objects|select(has("denom"))).denom           |= "akash"' | \
    jq -rM '(..|objects|select(has("bond_denom"))).bond_denom |= "akash"' | \
    jq -rM '(..|objects|select(has("mint_denom"))).mint_denom |= "akash"' > \
    "$NODE_DATA/config/genesis.json"

# add genesis accounts
gencoinamt="10000000akash"
akashd --home "$NODE_DATA" add-genesis-account $(akashctl --home $CLIENT_DATA keys show main -a) $gencoinamt
akashd --home "$NODE_DATA" add-genesis-account $(akashctl --home $CLIENT_DATA keys show provider -a) $gencoinamt
akashd --home "$NODE_DATA" add-genesis-account $(akashctl --home $CLIENT_DATA keys show validator -a) $gencoinamt
akashd --home "$NODE_DATA" add-genesis-account $(akashctl --home $CLIENT_DATA keys show other -a) $gencoinamt

# gentx and finalize genesis
akashd --home "$NODE_DATA" validate-genesis &> /dev/null
akashd --home "$NODE_DATA" --keyring-backend=test gentx --name validator --home-client "$CLIENT_DATA" --amount $gencoinamt &> /dev/null
akashd --home "$NODE_DATA" collect-gentxs &> /dev/null
akashd --home "$NODE_DATA" validate-genesis &> /dev/null

echo "Starting akashd instance..."
akashd --home $AKASH_DATA/node start --pruning=nothing > chain.log 2>&1 &
