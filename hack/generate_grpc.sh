#!/bin/bash

set -e
EMPTY_PROTO_VER=24.3.0

# check and download dependency for gRPC code generate
if [ ! -e ./grpc/proto/vendor/protobuf/src/google/protobuf ]; then
    DIR="./grpc/proto/vendor/protobuf/src/google/protobuf"
    rm -rf $DIR
    mkdir -p $DIR
    wget https://raw.githubusercontent.com/protocolbuffers/protobuf/v${EMPTY_PROTO_VER}/src/google/protobuf/empty.proto -P $DIR
fi

# gen-go-pb
protoc -I grpc/profiler/proto -I grpc/proto/vendor/protobuf/src/ grpc/profiler/proto/profiler.proto --go_out=plugins=grpc:./grpc