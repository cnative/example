#!/bin/sh
set -e

DIR="`dirname \"$0\"`"
ROOTDIR="`( cd \"$DIR/../\" && pwd )`"  # normalized project root dir

cd $ROOTDIR/web/build
$ROOTDIR/.tools/bin/go-bindata -o ../web.go -pkg web -nomemcopy ./...