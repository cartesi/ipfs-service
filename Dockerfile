FROM golang:1.14.7-alpine as build-image

RUN apk add --no-cache alpine-sdk git protoc

# get protoc-gen-go@1.42
RUN GO111MODULE=on go get github.com/golang/protobuf/protoc-gen-go@d04d7b157bb510b1e0c10132224b616ac0e26b17
# get protoc-gen-go-grpc@v1.32.0-dev
RUN GO111MODULE=on go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@5f7b337d951f2b7e13cdea854c5893ca940a0c77

ENV BASE /opt/cartesi
WORKDIR $BASE
        
# Download packages first so they can be cached.
COPY ./go.mod ./go.sum $BASE/
RUN go mod download

# Generating grpc-interfaces go files
# ----------------------------------------------------
COPY ./grpc-interfaces /root/grpc-interfaces
RUN \
    mkdir -p /root/grpc-interfaces/out \
    && cd /root/grpc-interfaces \
    && protoc \
        --go_out=./out \
        --go-grpc_out=./out \
        ipfs.proto

RUN mkdir -p $BASE/proto
RUN cp /root/grpc-interfaces/out/*.go $BASE/proto/

COPY ./server/ $BASE/server
RUN \
    cd ./server \
    && go build

# Emulator image, contains merkle-tree-hash util and its dependencies
FROM cartesi/machine-emulator:0.7.0-alpine as emulator

# Container final image
# starts from the same alpine version the buider-image above starts,
# because we only need golang to build
FROM alpine:3.12

RUN apk add --no-cache libstdc++

ENV BASE /opt/cartesi
WORKDIR $BASE

RUN mkdir -p $BASE/bin
RUN mkdir -p $BASE/lib

COPY --from=emulator $BASE/lib/libcryptopp* $BASE/lib/
COPY --from=emulator $BASE/bin/merkle-tree-hash $BASE/bin/
COPY --from=build-image $BASE/server/server $BASE/bin/

ENTRYPOINT ["/opt/cartesi/bin/server"]
