# IPFS Service

Cartesi IPFS service is a gRPC server that implements the [`AddFile`](#addfile) and [`GetFile`](#getfile) endpoints and performs them using an external IPFS node. The endpoints are defined in the submodule `grpc-interfaces/ipfs.proto`, and are required to access the filesystem where the files are being manipulated. A gRPC server and test client are implemented in Golang in the `ipfs-api-server` and `test_client` directories.

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
- Golang 1.14
- [Go support for Protocol Buffers v1.42](https://github.com/golang/protobuf/tree/master/protoc-gen-go)
  - `go get github.com/golang/protobuf/protoc-gen-go@d04d7b157bb510b1e0c10132224b616ac0e26b17`
- [Go plugin for Protocol Buffers v1.32-dev](https://github.com/grpc/grpc-go/tree/master/cmd/protoc-gen-go-grpc)
  - `go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@5f7b337d951f2b7e13cdea854c5893ca940a0c77`
- [grpc interfaces](https://github.com/cartesi/grpc-interfaces)

### Compile proto file to go source

Run the script to compile proto file:
```bash
$ ./compile.sh
```

### Executing the gRPC server

Install the Cartesi Emulator such that /opt/cartesi/bin/merkle-tree-hash exists

Run a local IPFS node, for example downloading [Kubo](https://github.com/ipfs/kubo)

Initialize the node (if not already done) & start up the local IPFS node
```
ipfs init --profile=server

ipfs daemon &
```

Change directory to `ipfs-api-server` and execute from there:
```bash
$ cd ipfs-api-server
$ go run main.go -g http://localhost:5001
```

You can also point the parameter at an IPFS API separate from localhost

### Executing the test client

Execute the gRPC server as in previous step

Change directory to `test_client` and execute from there:
```bash
$ cd test_client
$ go run main.go
```

Sample correct test run:
```
2022/12/19 20:48:10 testing GetFile error
2022/12/19 20:48:10 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:10 GetFile: &{updated_at:1671479290}
2022/12/19 20:48:11 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:11 GetFile: &{updated_at:1671479291}
2022/12/19 20:48:12 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:12 GetFile: &{updated_at:1671479292}
2022/12/19 20:48:13 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:13 GetFile: &{updated_at:1671479293}
2022/12/19 20:48:14 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:14 GetFile: &{updated_at:1671479294}
2022/12/19 20:48:15 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:15 could not GetFile: rpc error: code = Unknown desc = Could not get file: Post "http://localhost:5001/api/v0/get?arg=%!F(MISSING)ipfs%!F(MISSING)QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp&create=true": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2022/12/19 20:48:15 testing AddFile error
2022/12/19 20:48:15 Received AddFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f9e0} sizeCache:0 unknownFields:[] FilePath:/tmp/ipfs-test-1671479295}
2022/12/19 20:48:15 AddFile: &{updated_at:1671479295}
2022/12/19 20:48:16 Received AddFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f9e0} sizeCache:0 unknownFields:[] FilePath:/tmp/ipfs-test-1671479295}
2022/12/19 20:48:16 could not AddFile: rpc error: code = Unknown desc = Could not add file: lstat /tmp/ipfs-test-1671479295: no such file or directory
2022/12/19 20:48:16 testing AddFile
2022/12/19 20:48:16 Received AddFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f9e0} sizeCache:0 unknownFields:[] FilePath:/tmp/ipfs-test-1671479295}
2022/12/19 20:48:16 AddFile: &{updated_at:1671479296}
2022/12/19 20:48:16 Added file to IPFS with CID QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH
2022/12/19 20:48:17 Received AddFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f9e0} sizeCache:0 unknownFields:[] FilePath:/tmp/ipfs-test-1671479295}
2022/12/19 20:48:17 AddFile: &{ipfs_path:"QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH"}
2022/12/19 20:48:17 testing GetFile
2022/12/19 20:48:17 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:17 GetFile: &{updated_at:1671479297}
2022/12/19 20:48:17 Fetched file from IPFS: QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH -> /tmp/outout, safe name /tmp/outout_1274bb5509a85f763e7236b2274046261948f5903c2feb99019df50fa8be7ec8
2022/12/19 20:48:18 Received GetFileRequest: {state:{NoUnkeyedLiterals:{} DoNotCompare:[] DoNotCopy:[] atomicMessageInfo:0xc00015f500} sizeCache:0 unknownFields:[] IpfsPath:QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH Log2Size:12 OutputPath:/tmp/outout Timeout:5}
2022/12/19 20:48:18 GetFile: &{output_path:"/tmp/outout_1274bb5509a85f763e7236b2274046261948f5903c2feb99019df50fa8be7ec8"  root_hash:{data:"عn[oE\x9e\x9c\xb6\xa2\xf4\x1b\xf2vǸ\\\x10\xcdFb\xc0L\xbb\xb3eCG&\xc0\xa0"}}
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
