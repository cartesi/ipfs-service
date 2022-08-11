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
	"flag"
	"log"
	"os"
	"time"

	pb "github.com/cartesi/ipfs-service/proto"

	"google.golang.org/grpc"
)

func getFile(ctx context.Context, in *pb.GetFileRequest, c *pb.IpfsClient) (*pb.GetFileResponse, error) {
	for {
		r, err := (*c).GetFile(ctx, in)
		if err != nil {
			return nil, err
		}
		log.Printf("GetFile: %s", r.GetGetOneof())

		switch r.GetGetOneof().(type) {
		case *pb.GetFileResponse_Result:
			return r, nil
		}

		time.Sleep(time.Second)
	}
}

func addFile(ctx context.Context, in *pb.AddFileRequest, c *pb.IpfsClient) (*pb.AddFileResponse, error) {
	for {
		r, err := (*c).AddFile(ctx, in)
		if err != nil {
			return nil, err
		}
		log.Printf("AddFile: %s", r.GetAddOneof())

		switch r.GetAddOneof().(type) {
		case *pb.AddFileResponse_Result:
			return r, nil
		}

		time.Sleep(time.Second)
	}
}

func main() {
	// Parse command line flags
	addressPtr := flag.String("address", "localhost:50051", "server address")
	modePtr := flag.String("mode", "test", "client mode(add, get, or test)")
	argPtr := flag.String("argument", "", "file path for add, ipfs path for get, blank for test")

	flag.Parse()

	// Set up a connection to the server.
	conn, err := grpc.Dial(*addressPtr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewIpfsClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	switch *modePtr {
	case "add":
		addRequest := pb.AddFileRequest{
			FilePath: *argPtr,
		}

		r, err := addFile(ctx, &addRequest, &c)

		if err != nil {
			log.Fatalf("could not AddFile: %v", err)
		} else {
			switch r.GetAddOneof().(type) {
			case *pb.AddFileResponse_Progress:
				log.Fatalf("AddFile should return result")
			}
		}
	case "get":
		getRequest := pb.GetFileRequest{
			IpfsPath:   *argPtr,
			Log2Size:   0,
			OutputPath: "/tmp/outout",
			Timeout:    30,
		}

		r, err := getFile(ctx, &getRequest, &c)

		if err != nil {
			log.Fatalf("could not GetFile: %v", err)
		} else {
			switch r.GetGetOneof().(type) {
			case *pb.GetFileResponse_Progress:
				log.Fatalf("GetFile should return result")
			}
		}
	case "test":
		// Test GetFile error when the file doesn't exist
		// Should return timeout error after 5 seconds
		log.Printf("testing GetFile error")
		getRequest := pb.GetFileRequest{
			IpfsPath:   "/ipfs/QmWtCNv1euC7Fqkv61npo8LqrPLp3sVpsQHHj2dqg7Ljwp",
			Log2Size:   10,
			OutputPath: "/tmp/outout",
			Timeout:    5,
		}

		getResponse, err := getFile(ctx, &getRequest, &c)

		if err != nil {
			log.Printf("could not GetFile: %v", err)
		} else {
			switch getResponse.GetGetOneof().(type) {
			case *pb.GetFileResponse_Result:
				log.Fatalf("GetFile should fail")
			}
		}

		testFileName := "/tmp/ipfs-test"

		_, err = os.Stat(testFileName)
		if !os.IsNotExist(err) {
			// Remove the test file if exists
			err = os.Remove(testFileName)
			if err != nil {
				log.Fatalf("could not remove test file %s", err)
			}
		}

		// Test AddFile error when the file doesn't exist
		log.Printf("testing AddFile error")
		addRequest := pb.AddFileRequest{
			FilePath: testFileName,
		}

		addResponse, err := addFile(ctx, &addRequest, &c)

		if err != nil {
			log.Printf("could not AddFile: %v", err)
		} else {
			switch addResponse.GetAddOneof().(type) {
			case *pb.AddFileResponse_Result:
				log.Fatalf("AddFile should fail")
			}
		}

		_, err = os.Stat(testFileName)
		if os.IsNotExist(err) {
			// Create the test file if not exist
			file, err := os.Create(testFileName)
			if err != nil {
				log.Fatalf("could not create test file %s", err)
			}
			defer file.Close()
		}

		// Test AddFile with progress until done
		log.Printf("testing AddFile")

		addResponse, err = addFile(ctx, &addRequest, &c)

		if err != nil {
			log.Fatalf("could not AddFile: %v", err)
		} else {
			switch addResponse.GetAddOneof().(type) {
			case *pb.AddFileResponse_Progress:
				log.Fatalf("AddFile should return result")
			}
		}

		// Test GetFile with progress until done
		log.Printf("testing GetFile")

		getRequest.IpfsPath = addResponse.GetResult().GetIpfsPath()

		getResponse, err = getFile(ctx, &getRequest, &c)

		if err != nil {
			log.Fatalf("could not GetFile: %v", err)
		} else {
			switch getResponse.GetGetOneof().(type) {
			case *pb.GetFileResponse_Progress:
				log.Fatalf("GetFile should return result")
			}
		}
	}
}
