module github.com/ovrclk/deploy

go 1.14

require (
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/cosmos/go-bip39 v0.0.0-20180819234021-555e2067c45d
	github.com/ovrclk/akash v0.9.0-rc13
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/tendermint/tendermint v0.34.0-rc6
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
