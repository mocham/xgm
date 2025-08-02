#!/bin/bash

case "$1" in
    cli)
        go build -tags netgo -o Bin/cli cli.go;;
    root)
        go build -tags netgo -o Bin/root_x root.go;;
    *)
        echo "Error: Unknown script '$1'";;
esac
