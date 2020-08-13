# IPFS Service

Cartesi IPFS service is a gRPC server that implements the [`AddFile`](#addfile) and [`GetFile`](#getfile) endpoints for its in-process IPFS node. The endpoints are defined in the submodule `grpc-interfaces/ipfs.proto`, and are required to access the filesystem where the files are being manipulated. A gRPC server and test client are implemented in Golang in the `server` and `test_client` directories.

## AddFile

The request body of `AddFile` simply contains the path to the file to be added. A gRPC error will be thrown if the file path is not valid or not reachable from the server process.
```
message AddFileRequest {
    string file_path = 1;
}
```
And the response is either a `Progress` which contains current progress percentage and the latest update time if the action is not complete;
```
message Progress {
    uint64 progress = 1;
    uint64 updated_at = 2;
}
```
Or the `AddFileResult` which contains the IPFS path of the added file.
```
message AddFileResult {
    string ipfs_path = 1;
}
```
```
message AddFileResponse {
    oneof add_oneof {
        Progress progress = 1;
        AddFileResult result = 2;
    }
}
```

## GetFile

The request body of `GetFile` contains the following values:
- `ipfs_path`: the IPFS path of the file to be retrieved
- `log2_size`: the expected file size in the log2 representation of bytes unit, server will check the size of the retrieved file, a gRPC error will be thrown if file size exceeds the expected log2 size (specifying `0` will disable this check, and result in `0`s in the merkle tree root hash in response)
- `output_path`: the path of the file to be stored after successful retrival from IPFS node
- `timeout`: the timeout in seconds of the request, a gRPC error will be thrown if the timeout is reached before a successful file retrival (specifying `0` will use the server's default timeout, which is one hour)
```
message GetFileRequest {
    string ipfs_path = 1;
    uint32 log2_size = 2;
    string output_path = 3;
    uint64 timeout = 4;
}
```
And the response is either a `Progress` which contains current progress percentage and the latest update time if the action is not complete;
```
message Progress {
    uint64 progress = 1;
    uint64 updated_at = 2;
}
```
Or the `GetFileResult` which contains the output path of the retrieved file, and the file's merkle tree root hash calculated from the log2 size specified in the request.
```
message GetFileResult {
    string output_path = 1;
    Hash root_hash = 2;
}
```
```
message GetFileResponse {
    oneof get_oneof {
        Progress progress = 1;
        GetFileResult result = 2;
    }
}
```

*Note that the merkle tree naturally requires the file to be power of 2 size, if not, will zero-padding to the next power of 2 size.*

## Quick Start

[PLACEHOLDER]

### Requirements

- [Protocol Buffers](https://github.com/protocolbuffers/protobuf)
  - `apt-get install protobuf-compiler`
- Golang >= 1.14
- [Go support for Protocol Buffers](https://github.com/golang/protobuf/tree/master/protoc-gen-go)
  - `go get github.com/golang/protobuf/protoc-gen-go`
- [Go plugin for Protocol Buffers](https://github.com/grpc/grpc-go/tree/master/cmd/protoc-gen-go-grpc)
  - `go get google.golang.org/grpc/cmd/protoc-gen-go-grpc`
- [grpc interfaces](https://github.com/cartesi/grpc-interfaces)

### Compile proto file to go source

Run the script to compile proto file:
```bash
$ ./compile.sh
```

### Executing the gRPC server

Change directory to `server` and execute from there:
```bash
$ cd server
$ go run main.go
```

### Executing the test client

Change directory to `test_client` and execute from there:
```bash
$ cd test_client
$ go run main.go
```

### With docker

[PLACEHOLDER]

## Contributing

Pull requests are welcome. When contributing to this repository, please first discuss the change you wish to make via issue, email, or any other method with the owners of this repository before making a change.

Please note we have a code of conduct, please follow it in all your interactions with the project.

## Authors

* *Stephen Chen*

## License

[PLACEHOLDER]

## Acknowledgments

- [Use go-ipfs as a library](https://github.com/ipfs/go-ipfs/tree/master/docs/examples/go-ipfs-as-a-library)
- [gRPC Hello World](https://github.com/grpc/grpc-go/tree/master/examples/helloworld)
