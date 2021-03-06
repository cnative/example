#!/bin/bash
set -e

(
    ROOTDIR=$(dirname $PWD)/..
    GW_ROOT_DIR="$ROOTDIR/vendor/github.com/grpc-ecosystem/grpc-gateway"
    protoc="$ROOTDIR/.tools/bin/protoc -I./proto -I$ROOTDIR/.tools/include -I$GW_ROOT_DIR/third_party/googleapis -I$GW_ROOT_DIR"

    cd $ROOTDIR

    $protoc --go_out=paths=source_relative,plugins=grpc:$ROOTDIR/pkg/api \
            --grpc-gateway_out=logtostderr=true,request_context=true:$ROOTDIR/pkg/api \
            --swagger_out=logtostderr=true:$ROOTDIR/pkg/api \
            --plugin=protoc-gen-go=$ROOTDIR/.tools/bin/protoc-gen-go \
            --plugin=protoc-gen-grpc-gateway=$ROOTDIR/.tools/bin/protoc-gen-grpc-gateway \
            --plugin=protoc-gen-swagger=$ROOTDIR/.tools/bin/protoc-gen-swagger \
        proto/report.proto
)
