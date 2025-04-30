default:
    @just --list

test:
    go test -fullpath ./lib

build:
    go build

install:
    go build -o ~/.local/bin/hm
