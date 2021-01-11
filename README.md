## Hub CLI

Hub CLI is stack composition and lifecycle tool.

    $ hub elaborate hub.yaml params.yaml -o hub.yaml.elaborate
    $ hub deploy hub.yaml.elaborate -e NAME=test2
    $ hub version
    $ hub help

## Build

Use `make` to build Hub CLI:

    $ make

Or directly with `go`:

    $ go get github.com/agilestacks/hub/cmd/hub

## Clean up

    $ make clean

## What's next?

Deploy [App Stack](https://github.com/agilestacks/stack-app-eks) or [ML Stack](https://github.com/agilestacks/stack-ml-eks) on EKS.

More information in the [wiki](https://github.com/agilestacks/hub/wiki) and [docs](https://docs.agilestacks.com).
