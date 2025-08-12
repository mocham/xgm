#!/bin/bash
DIMG="make"

go() {
    docker run -it --rm \
        -e GOPATH="/go" \
        -e PATH="/usr/bin/path:/usr/bin:/home/linuxbrew/.linuxbrew/bin" \
        -v $HOME/.local/go:/go:rw \
        -v $HOME/.cache/go-build:/root/.cache/go-build:rw \
        -v /opt/Runtime/Docker-Home/go:/root/.config/go:rw \
        -v $PWD:/root/Go \
        --workdir /root/Go \
        --entrypoint "/usr/bin/go" \
        -e https_proxy=$http_proxy \
        $DIMG build -tags netgo -o $@
}

case "$1" in
    cli)
        go cli cli*.go
        mv cli $HOME/Bar/Go/Bin/cli
        ;;
    root)
        go root_x root.go
        mv root_x $HOME/Bar/Go/Bin/root_x
        ;;
    *)
        echo "Error: Unknown script '$1'";;
esac
