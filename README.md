## Hub CLI

    $ hub elaborate hub.yaml params.yaml -o hub.yaml.elaborate
    $ hub deploy hub.yaml.elaborate -e NAME=test2
    $ hub version
    $ hub help

## Build

Use `make` to build Hub CLI:

    $ make get

Or directly with `go`:

    $ go get github.com/agilestacks/hub/cmd/hub

## Clean up

    $ make clean
