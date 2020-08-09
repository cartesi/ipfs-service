// Copyright 2019 Cartesi Pte. Ltd.

// Licensed under the Apache License, Version 2.0 (the "License"); you may not use
// this file except in compliance with the License. You may obtain a copy of the
// License at http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package main implements a test client for Ipfs service.
package main

import (
	"context"
	"log"
	"time"

	pb "ipfs/proto"

	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewIpfsClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test GetFile error when the file doesn't exist
	// Should return timeout error after 5 seconds
	log.Printf("testing GetFile error")
	for {
		r, err := c.GetFile(ctx, &pb.GetFileRequest{
			IpfsPath:   "/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp",
			Log2Size:   10,
			OutputPath: "/tmp/outout",
			Timeout:	5,
		})
		if err != nil {
			log.Printf("could not get: %v", err)
			break
		}
		log.Printf("GetFile: %s", r.GetGetOneof())

		switch r.GetGetOneof().(type) {
		case *pb.GetFileResponse_Result:
			log.Fatalf("GetFile should fail")
		}

		time.Sleep(time.Second)
	}

	// Test AddFile error when the file doesn't exist
	log.Printf("testing AddFile error")
	for {
		r, err := c.AddFile(ctx, &pb.AddFileRequest{
			FilePath: "/tmp/remote00-wsl-loc.txt",
		})
		if err != nil {
			log.Printf("could not add: %v", err)
			break
		}
		log.Printf("AddFile: %s", r.GetAddOneof())

		switch r.GetAddOneof().(type) {
		case *pb.AddFileResponse_Result:
			log.Fatalf("AddFile should fail")
		}

		time.Sleep(time.Second)
	}

	var ipfsPath string
	// Test AddFile with progress until done
	log.Printf("testing AddFile")
AddLoop:
	for {
		r, err := c.AddFile(ctx, &pb.AddFileRequest{
			FilePath: "/tmp/remote-wsl-loc.txt",
		})
		if err != nil {
			log.Fatalf("could not add: %v", err)
		}
		log.Printf("AddFile: %s", r.GetAddOneof())

		switch r.GetAddOneof().(type) {
		// Break the loop when result is ready
		case *pb.AddFileResponse_Result:
			ipfsPath = r.GetResult().GetIpfsPath()
			break AddLoop
		}

		time.Sleep(time.Second)
	}

	// Test GetFile with progress until done
	log.Printf("testing GetFile")
GetLoop:
	for {
		r, err := c.GetFile(ctx, &pb.GetFileRequest{
			IpfsPath:   ipfsPath,
			Log2Size:   10,
			OutputPath: "/tmp/outout",
			Timeout:	10,
		})
		if err != nil {
			log.Fatalf("could not get: %v", err)
		}
		log.Printf("GetFile: %s", r.GetGetOneof())

		switch r.GetGetOneof().(type) {
		// Break the loop when result is ready
		case *pb.GetFileResponse_Result:
			break GetLoop
		}

		time.Sleep(time.Second)
	}
}
