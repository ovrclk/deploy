# Deploy

Deploy is a prototype for exploring what it would look like to have a watcher daemon for user deployments.

To use it run the following:

Terminal 1
```bash
cd $GOPATH/src/github.com/ovrclk/akash/_run/kube
make clean init node-run
# let the node output stream here
```

Terminal 2
```bash
cd $GOPATH/src/github.com/ovrclk/akash/_run/kube
KIND_CONFIG=kind-config-80.yaml make kind-cluster-delete kind-cluster-create
# wait for it to finish
../../akashctl --home ./cache/client keys export main
# finish the prompts (pw: 12345678) and save the key output in your clipboard :shushing_face:
make provider-create provider-run
# let the provider logs stream here
```

Terminal 3
```bash
# from the root of this directory
mkdir ~/.akash-deploy && cat <<EOT >> ~/.akash-deploy/config.toml
chain-id: "local"
rpc-addr: "http://localhost:26657"
keyfile: "key.priv"
keypass: "12345678"
EOT

# paste the key output into the keyfile
pbpaste > ~/.akash-deploy/key.priv
```