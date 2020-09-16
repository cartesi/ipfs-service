// Copyright 2019 Cartesi Pte. Ltd.

// Licensed under the Apache License, Version 2.0 (the "License"); you may not use
// this file except in compliance with the License. You may obtain a copy of the
// License at http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package main implements a server for Ipfs service.
package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"
	"os"
	"path/filepath"
	"io/ioutil"
	"fmt"

	pb "ipfs/proto"
	
	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	port = ":50051"
)

// server is used to implement ipfs.IpfsServer.
type server struct {
	pb.UnimplementedIpfsServer
}

// request is used to track status of each incoming request
type request struct {
	done    chan string
	err     chan error
	running	bool
	result 	string
}

// addParams is used submit AddFile request to IPFS handler routine
type addParams struct {
	done    	chan string
	err     	chan error
	filePath 	string
}

// getParams is used submit GetFile request to IPFS handler routine
type getParams struct {
	done    	chan string
	err     	chan error
	ipfsPath 	string
	outputPath 	string
	timeout		uint64
}

// SafeMap is used to track status of each request
type SafeMap struct {
	status  map[string]*request
	mux 	sync.Mutex
	addCh	chan addParams
	getCh	chan getParams
}

var safeMap = SafeMap{
	status: make(map[string]*request),
	addCh:	make(chan addParams),
	getCh:	make(chan getParams),
}

/// ------ Setting up the IPFS Repo

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

/// ------ Spawning the node

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

// Spawns a node on the default repo location, if the repo exists
func spawnDefault(ctx context.Context) (icore.CoreAPI, error) {
	defaultPath, err := config.PathRoot()
	if err != nil {
		// shouldn't be possible
		return nil, err
	}

	if err := setupPlugins(defaultPath); err != nil {
		return nil, err

	}

	return createNode(ctx, defaultPath)
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context) (icore.CoreAPI, error) {
	if err := setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Temporary Repo
	repoPath, err := createTempRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Spawning an ephemeral IPFS node
	return createNode(ctx, repoPath)
}

// Connect to peers list
func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func getUnixfsFile(path string) (files.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	st, err := file.Stat()
	if err != nil {
		return nil, err
	}

	f, err := files.NewReaderPathFile(path, file, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

/// -------

// AddFile implements ipfs.IpfsServer
func (s *server) AddFile(ctx context.Context, in *pb.AddFileRequest) (*pb.AddFileResponse, error) {
	log.Printf("Received AddFileRequest: %+v", *in)

	safeMap.mux.Lock()

	var response *pb.AddFileResponse = nil
	var err error = nil
	key := in.GetFilePath()
	status := safeMap.status[key]

	if status != nil {
		// Request being processed already
		if !status.running {
			// Return result as request is done
			response = &pb.AddFileResponse{
				AddOneof: &pb.AddFileResponse_Result{
					Result: &pb.AddFileResult{
						IpfsPath: status.result,
						}}}
		} else {
			// Pull progress or result as still running
			select {
			case status.result = <- status.done:
				// Return result
				response = &pb.AddFileResponse{
					AddOneof: &pb.AddFileResponse_Result{
						Result: &pb.AddFileResult{
							IpfsPath: status.result,
							}}}
				status.running = false
			case retErr := <-status.err:
				// Return error
				err = grpc.Errorf(codes.Unknown, retErr.Error())
				safeMap.status[key] = nil
			default:
				// Return progress
				response = &pb.AddFileResponse{
					AddOneof: &pb.AddFileResponse_Progress{
						Progress: &pb.Progress{
							// TODO: Calculate progress
							Progress:  0,
							UpdatedAt: uint64(time.Now().Unix()),
						}}}
			}
		}
	} else {
		// First time receive request
		safeMap.status[key] = &request{
			done:    make(chan string),
			err:     make(chan error),
			running: true,
		}

		go func() {
			// Submit AddFile job to IPFS
			safeMap.addCh <- addParams{
				done: 		safeMap.status[key].done,
				err: 		safeMap.status[key].err,
				filePath: 	key,
			}
		}()

		response = &pb.AddFileResponse{
			AddOneof: &pb.AddFileResponse_Progress{
				Progress: &pb.Progress{
					Progress:  0,
					UpdatedAt: uint64(time.Now().Unix()),
				}}}
	}

	safeMap.mux.Unlock()

	return response, err
}

// GetFile implements ipfs.IpfsServer
func (s *server) GetFile(ctx context.Context, in *pb.GetFileRequest) (*pb.GetFileResponse, error) {
	log.Printf("Received GetFileRequest: %+v", *in)

	safeMap.mux.Lock()

	var response *pb.GetFileResponse = nil
	var err error = nil
	key := in.GetIpfsPath()
	status := safeMap.status[key]

	if status != nil {
		// Request being processed already
		if !status.running {
			// Return result as request is done
			response = &pb.GetFileResponse{
				GetOneof: &pb.GetFileResponse_Result{
					Result: &pb.GetFileResult{
						OutputPath: status.result,
						RootHash: 	&pb.Hash{
							// TODO: Calculate Merkle root hash from file
							Data: make([]byte, 32),
						}}}}
		} else {
			// Pull progress or result as still running
			select {
			case status.result = <- status.done:
				// Return result
				response = &pb.GetFileResponse{
					GetOneof: &pb.GetFileResponse_Result{
						Result: &pb.GetFileResult{
							OutputPath: status.result,
							RootHash: 	&pb.Hash{
								// TODO: Calculate Merkle root hash from file
								Data: make([]byte, 32),
							}}}}
				status.running = false
			case retErr := <-status.err:
				// Return error
				err = grpc.Errorf(codes.Unknown, retErr.Error())
				safeMap.status[key] = nil
			default:
				// Return progress
				response = &pb.GetFileResponse{
					GetOneof: &pb.GetFileResponse_Progress{
						Progress: &pb.Progress{
							// TODO: Calculate progress
							Progress:  0,
							UpdatedAt: uint64(time.Now().Unix()),
						}}}
			}
		}
	} else {
		// First time receive request
		safeMap.status[key] = &request{
			done:    make(chan string),
			err:     make(chan error),
			running: true,
		}

		go func() {
			// Submit GetFile job to IPFS
			safeMap.getCh <- getParams{
				done:		safeMap.status[key].done,
				err: 		safeMap.status[key].err,
				ipfsPath: 	key,
				outputPath: in.GetOutputPath(),
				timeout:	in.GetTimeout(),
			}
		}()

		response = &pb.GetFileResponse{
			GetOneof: &pb.GetFileResponse_Progress{
				Progress: &pb.Progress{
					Progress:  0,
					UpdatedAt: uint64(time.Now().Unix()),
				}}}
	}

	safeMap.mux.Unlock()

	return response, err
}

func main() {
	/// --- Part I: Getting a IPFS node running

	log.Printf("-- Getting an IPFS node running -- ")

	ipfsReady := make(chan bool)

	go func() {
		ipfsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
	
		// Spawn a node using a temporary path, creating a temporary repo for the run
		log.Printf("Spawning node on a temporary repo")

		ipfs, err := spawnEphemeral(ipfsCtx)
		if err != nil {
			panic(fmt.Errorf("failed to spawn ephemeral node: %s", err))
		}

		log.Printf("IPFS node is running")

		log.Printf("-- Going to connect to a few nodes in the Network as bootstrappers --")

		bootstrapNodes := []string{
			// IPFS Bootstrapper nodes.
			// "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			// "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			// "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			// "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
			"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	
			// IPFS Cluster Pinning nodes
			"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
			"/ip4/138.201.67.219/udp/4001/quic/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
			"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
			"/ip4/138.201.67.220/udp/4001/quic/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
			"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
			"/ip4/138.201.68.74/udp/4001/quic/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
			"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
			"/ip4/94.130.135.167/udp/4001/quic/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
	
			// You can add more nodes here, for example, another IPFS node you might have running locally, mine was:
			// "/ip4/127.0.0.1/tcp/4010/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
			// "/ip4/127.0.0.1/udp/4010/quic/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
		}
	
		go connectToPeers(ipfsCtx, ipfs, bootstrapNodes)

		ipfsReady <- true

		for {
			// Start listening incoming requests from gRPC client
			select {
			case add := <- safeMap.addCh:
				// Add file to IPFS
				addFile, err := getUnixfsNode(add.filePath)
				if err != nil {
					add.err <- fmt.Errorf("Could not access File: %s", err)
					break
				}
			
				cidFile, err := ipfs.Unixfs().Add(ipfsCtx, addFile)
				if err != nil {
					add.err <- fmt.Errorf("Could not add File: %s", err)
					break
				}
			
				log.Printf("Added file to IPFS with CID %s", cidFile.String())
				add.done <- cidFile.String()
			case get := <- safeMap.getCh:
				// Get file from IPFS and write to output path
				ipfsGetCtx, cancel:= context.WithTimeout(ipfsCtx, time.Duration(get.timeout) * time.Second)
				defer cancel()

				cidFile := icorepath.New(get.ipfsPath)

				rootNodeFile, err := ipfs.Unixfs().Get(ipfsGetCtx, cidFile)
				if err != nil {
					get.err <- fmt.Errorf("Could not get File: %s", err)
					break
				}
	
				err = files.WriteTo(rootNodeFile, get.outputPath)
				if err != nil {
					get.err <- fmt.Errorf("Could not write out the fetched CID: %s", err)
					break
				}
			
				log.Printf("Got file from IPFS and write to %s", get.outputPath)
				get.done <- get.outputPath
			}
		}
	}()

	// Wait ipfs node to be ready before starting gRPC server
	<- ipfsReady
	
	/// --- Part II: Getting a gRPC server node running
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("gRPC server failed to listen: %v", err)
	} else {
		log.Printf("gRPC server started listening...")
	}
	s := grpc.NewServer()
	pb.RegisterIpfsServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed to serve: %v", err)
	}
}
