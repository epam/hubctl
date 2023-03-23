# hubctl

[hubctl](https://superhub.io) is a missing package manager for the cloud.

## Example of usage

```shell
hubctl elaborate hub.yaml params.yaml -o hub.yaml.elaborate
hubctl deploy hub.yaml.elaborate -e NAME=stage
hubctl version
hubctl help
```

## Installation

### Pre-build binary

To download the latest release, run:

#### cURL

```shell
curl -LJ "https://github.com/epam/hubctl/releases/latest/download/hubctl_$(uname -s)_$(uname -m).tar.gz" \|
  tar xz -C /tmp && sudo mv /tmp/hubctl /usr/local/bin
```

#### Wget

```shell
wget "https://github.com/epam/hubctl/releases/latest/download/hubctl_$(uname -s)_$(uname -m).tar.gz" -O - |\
  tar xz -C /tmp && sudo mv /tmp/hubctl /usr/local/bin
```

#### Other pre-build binaries

There are [macOS amd64](https://github.com/epam/hubctl/releases/latest/download/hubctl_Darwin_arm64.tar.gz), [macOS arm64](https://github.com/epam/hubctl/releases/latest/download/hubctl_Darwin_x86_64.tar.gz), [Linux amd64](https://github.com/epam/hubctl/releases/latest/download/hubctl_Linux_arm64.tar.gz), [Linux arm64](https://github.com/epam/hubctl/releases/latest/download/hubctl_Linux_x86_64.tar.gz) and [Windows x64](https://github.com/epam/hubctl/releases/latest/download/hubctl_Windows_x86_64.zip) binaries.

### [Homebrew](https://brew.sh/) Formula

```shell
brew tap epam/tap
brew install epam/tap/hubctl
```

### Extensions

Optionally, install extensions:

```shell
hubctl extensions install
```

Hub CTL Extensions requires: [bash], [curl], [jq] and [yq v4].
Optionally install [Node.js] and NPM for `hubctl pull` extension, [AWS CLI], [Azure CLI], [GCP SDK] [kubectl], [eksctl] for `hubctl ext eks` extension.

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

### Setup

Before make any changes you should configure git hooks for this repository

```shell
git config core.hooksPath .githooks
```

### Build

Use `make` to build Hub CTL:

```shell
make
```

Or directly with `go`:

```shell
go build -o bin/$(go env GOOS)/hubctl github.com/epam/hubctl/cmd/hub
```

### Clean up

```shell
make clean
```

## Usage metrics

When you use a pre-built binary from the releases page, it will send usage metrics to HubCTL API and Datadog.

We value your privacy and only send anonymized usage metrics for the following commands:

- elaborate
- deploy
- undeploy
- backup create
- api *

A usage metric sample contains:

- Hub CTL command invoked without arguments, ie. 'deploy' or 'backup create', or 'api instance get'
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

You could always review an up-to-date help via `hubctl util metrics -h`.

## What's next?

More information in the [wiki](https://github.com/epam/hubctl/wiki).

[AWS CLI]: https://aws.amazon.com/cli/
[Azure CLI]: https://docs.microsoft.com/en-us/cli/azure/
[GCP SDK]: https://cloud.google.com/sdk/docs/install
[kubectl]: https://kubernetes.io/docs/reference/kubectl/overview/
[eksctl]: https://eksctl.io
[jq]: https://stedolan.github.io/jq/
[yq v4]: https://github.com/mikefarah/yq
[Node.js]: https://nodejs.org
[bash]: https://www.gnu.org/software/bash
[curl]: https://curl.se
