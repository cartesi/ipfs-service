FROM golang:1.14.4-buster as build-image

ENV PATH="$PATH:$(go env GOPATH)/bin"

RUN \
    apt-get update && \
    apt-get install --no-install-recommends -y cmake protobuf-compiler && \
    rm -rf /var/lib/apt/lists/*

ENV GO111MODULE auto
RUN go get github.com/golang/protobuf/protoc-gen-go
RUN go get google.golang.org/grpc/cmd/protoc-gen-go-grpc

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

# Container final image
# ----------------------------------------------------
FROM golang:1.14.4-buster

ENV BASE /opt/cartesi
WORKDIR $BASE

# Install deps
RUN apt-get update && apt-get install -y \
  libssl-dev \
  ca-certificates \
  fuse

RUN mkdir -p $BASE/bin

COPY --from=build-image $BASE/server/server $BASE/bin

ENTRYPOINT ["/opt/cartesi/bin/server"]
