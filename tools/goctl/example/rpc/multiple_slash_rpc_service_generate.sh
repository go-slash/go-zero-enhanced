#!/bin/bash

wd=$(pwd)
output="$wd/slash"

rm -rf $output

goctls rpc protoc -I $wd "$wd/slash.proto" --go_out="$output/pb" --go-grpc_out="$output/pb" --zrpc_out="$output" --group -style=go_zero

if [ $? -ne 0 ]; then
    echo "Generate failed"
    exit 1
fi

GOPROXY="https://goproxy.cn,direct" && go mod tidy

if [ $? -ne 0 ]; then
    echo "Tidy failed"
    exit 1
fi

go test ./...