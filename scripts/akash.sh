#!/bin/bash

AKASH_REPO="$GOPATH/src/github.com/ovrclk/akash"
AKASH_BRANCH="v0.7.7"
CHAIN_ID="testchain"

# Ensure gopath is set and go is installed
if [[ ! -d $GOPATH ]] || [[ ! -d $GOBIN ]] || [[ ! -x "$(which go)" ]]; then
  echo "Your \$GOPATH is not set or go is not installed,"
  echo "ensure you have a working installation of go before trying again..."
  echo "https://golang.org/doc/install"
  exit 1
fi

# ARGS: 
# $1 -> local || remote, defaults to remote

set -e

if [[ -d $AKASH_REPO ]]; then
  cd $AKASH_REPO

  # remote build syncs with remote then builds
  if [[ "$1" == "local" ]]; then
    echo "Installing local branch of github.com/ovrclk/akash..."
    make install &> /dev/null
  else
    echo "Building github.com/ovrclk/akash@$AKASH_BRANCH..."
    if [[ ! -n $(git status -s) ]]; then
      # sync with remote $AKASH_BRANCH
      git fetch --all &> /dev/null

      # ensure the akash repository successfully pulls the latest $AKASH_BRANCH
      if [[ -n $(git checkout $AKASH_BRANCH -q) ]] || [[ -n $(git pull origin $AKASH_BRANCH -q) ]]; then
        echo "failed to sync remote branch $AKASH_BRANCH"
        echo "in $AKASH_REPO, please rename the remote repository github.com/ovrclk/akash to 'origin'"
        exit 1
      fi

      # install
      make install &> /dev/null

      # ensure that built binary has the same version as the repo
      if [[ ! "$(akashd version --long 2>&1 | grep "commit:" | sed 's/commit: //g')" == "$(git rev-parse HEAD)" ]]; then
        echo "built version of akashd commit doesn't match"
        exit 1
      fi 
    else
      echo "uncommited changes in $AKASH_REPO, please commit or stash before building"
      exit 1
    fi
    
  fi 
else 
  echo "$AKASH_REPO doesn't exist, and you may not have have the akash repo locally,"
  echo "if you want to download akash to your \$GOPATH try running the following command:"
  echo "mkdir -p $(dirname $AKASH_REPO) && git clone git@github.com:cosmos/akash $AKASH_REPO"
fi

