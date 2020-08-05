# Deploy

> NOTE: :dragon: WIP :dragon: Please :dragon: Open :dragon: Issues :dragon: You :dragon: Find :dragon:

`deploy` is a command line client for deploying applications on the [Akash Network](https://akash.network). It also contains a full demo environment to help users develop their [SDL files](https://docs.akash.network/usage/sdl) for deployment on the live network (test or otherwise).

### Requirements

* Go 1.14+ installed and `$GOPATH` + `$GOBIN` setup
    - [Install Go](https://golang.org/doc/install)

### Creating your first Akash application

The following commands will deploy your first akash application on the testnet:

```bash
# First, if you haven't, install the `deploy` binary
make install

# Next, generate the configuration file for the testnet
deploy init testnet-v4 http://rpc.akashtest.net:26657

# Create a private key for your deployments...
deploy key-add

# ...get the address for the key you just created...
deploy address

# ...and take it to the faucet: https://akash.vitwit.com/faucet
# when you have tokens, you will see them using the balance command
deploy balance

# Once you have some testnet `akash` you can start deploying apps!
# Try the `sample.yaml` file in the root of the repo...
deploy create sample.yaml
```