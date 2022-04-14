# Hub CLI

[Hub CLI](https://superhub.io) is stack composition and lifecycle tool.

## Example of usage

```shell
hub elaborate hub.yaml params.yaml -o hub.yaml.elaborate
hub deploy hub.yaml.elaborate -e NAME=stage
hub version
hub help
```

## Installation

There are [macOS amd64](https://github.com/agilestacks/hub/releases/download/v1.0.5/hub.darwin_amd64), [macOS arm64](https://github.com/agilestacks/hub/releases/download/v1.0.5/hub.darwin_arm64), [Linux amd64](https://github.com/agilestacks/hub/releases/download/v1.0.5/hub.linux_amd64), [Linux arm64](https://github.com/agilestacks/hub/releases/download/v1.0.5/hub.linux_arm64) and [Windows x64](https://github.com/agilestacks/hub/releases/download/v1.0.5/hub.windows_amd64.exe) binaries.

### Pre-build binary

```shell
VERSION=v1.0.5
BINARY=hub.darwin_amd64
curl -L -O https://github.com/agilestacks/hub/releases/download/$VERSION/$BINARY
mv $BINARY hub
chmod +x hub
sudo mv hub /usr/local/bin
```

### Homebrew

```shell
brew tap agilestacks/brew
brew install agilestacks/brew/hub
```

Sorry for the name conflict with [GitHub hub](https://hub.github.com).

### Extensions

Optionally, install extensions:

```shell
hub extensions install
```

Hub CLI Extensions require [AWS CLI] or [Azure CLI], [kubectl], [jq], [yq v4]. Optionally install [Node.js] and NPM for `hub pull` extension. Optionally install [eksctl] for `hub ext eks` extension.

### macOS users

Depending on your's machine Security & Privacy settings and macOS version (10.15+), you may get an error _cannot be opened because the developer cannot be verified_. Please [read on](https://github.com/hashicorp/terraform/issues/23033#issuecomment-542302933) for a simple workaround:

```shell
xattr -d com.apple.quarantine hub.darwin_amd64
```

Alternatively, to set a global preference to _Allow apps downloaded from: Anywhere_, execute:

```shell
sudo spctl --master-disable
```

## Development

### Build

Use `make` to build Hub CLI:

```shell
make
```

Or directly with `go`:

```shell
go get github.com/agilestacks/hub/cmd/hub
```

### Clean up

```shell
make clean
```

## Usage metrics

When you use a pre-built binary from the releases page, it will send usage metrics to SuperHub and Datadog.

We value your privacy and only send anonymized usage metrics for the following commands:

- elaborate
- deploy
- undeploy
- backup create
- api *

A usage metric sample contains:

- Hub CLI command invoked without arguments, ie. 'deploy' or 'backup create', or 'api instance get'
- synthetic machine id - an UUID generated in first interactive session (stdout is a TTY)
- usage counter - 1 per invocation

Edit `$HOME/.hub-cache.yaml` to change settings:

```yaml
    metrics:
      disabled: false
      host: 68af657e-6a51-4d4b-890c-4b548852724d
```

Set `disabled: true` to skip usage metrics reporting.
Set `host: ""` to send the counter but not the UUID.

You could always review an up-to-date help via `hub util metrics -h`.

## What's next?

Deploy [App Stack](https://github.com/agilestacks/stack-app-eks) or [Machine Learning Stack](https://github.com/agilestacks/stack-ml-eks) on AWS EKS.

More information in the [wiki](https://github.com/agilestacks/hub/wiki).

[AWS CLI]: https://aws.amazon.com/cli/
[Azure CLI]: https://docs.microsoft.com/en-us/cli/azure/
[kubectl]: https://kubernetes.io/docs/reference/kubectl/overview/
[eksctl]: https://eksctl.io
[jq]: https://stedolan.github.io/jq/
[yq v4]: https://github.com/mikefarah/yq
[Node.js]: https://nodejs.org
