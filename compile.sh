#!/bin/sh
mkdir -p proto
cd grpc-interfaces
mkdir -p out
protoc \
  --go_out=./out \
  --go-grpc_out=./out \
  ipfs.proto
cp out/*.go ../proto/
rm -rf out/
