#!/bin/sh

set -e
set -x

OUTDIR=$(dirname $0)/../out

if [ ! -d out ]; then
  mkdir out
fi

GOARCH=amd64 GOOS=linux go build -o rtr && mv rtr $OUTDIR/rtr-linux-amd64
GOARCH=386 GOOS=linux go build -o rtr && mv rtr $OUTDIR/rtr-linux-386
GOARCH=amd64 GOOS=darwin go build -o rtr && mv rtr $OUTDIR/rtr-darwin-amd64
