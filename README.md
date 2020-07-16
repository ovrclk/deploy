# Deploy

> NOTE: :dragon: WIP :dragon: Please :dragon: Open :dragon: Issues :dragon: You :dragon: Find :dragon:

`deploy` is a command line client for deploying applications on the [Akash Network](https://akash.network). It also contains a full demo environment to help users develop their [SDL files](https://docs.akash.network/usage/sdl) for deployment on the live network (test or otherwise).

### Requirements

* Docker installed and running
    - [Install Docker](https://docs.docker.com/get-docker/)
* Go 1.14+ installed and `$GOPATH` + `$GOBIN` setup
    - [Install Go](https://golang.org/doc/install)

### Running the demo environment

The demo environment sets up:
* A kubernetes cluster in docker using [`kind`](https://github.com/kubernetes-sigs/kind)
* A running [Akash chain instance](https://github.com/ovrclk/akash)

> NOTE: The kube cluster, especially on the first run pulls quite a bit of data locally. Depending on your connection this may take a while.

```bash
# First, if you haven't, install the dependancies
make install-deps

# Then start the demo environment
# NOTE: this can take a while, please wait for the command to finish
make demo

# Then you can start deploying apps!
# Try the `sample.yaml` file in the root of the repo...
deploy create sample.yaml

# You app will be available at: http://hello.localhost!
```

### TODOS:

- [ ] Give the deployments user generated names
- [ ] Add crud for the sdl file database
- [ ] Add git integration to allow for easy storage of the configuration directory of this repository
- [ ] Embed some sample deployments into the binary :thinking_face:
- [ ] Add management for full on-chain deployment lifecycle 
- [ ] Queries for some state?
