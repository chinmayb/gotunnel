package grpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/chinmayb/gotunnel/pkg/pb"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

func grpcHandler() error {
	grpcServerEndpoint := "localhost:50051"

	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				unaryInterceptors...)),
	)

	pb.RegisterTunnelServer(grpcServer, &server{})

	grpcL, err := net.Listen("tcp", grpcServerEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Starting gRPC server on %s", grpcServerEndpoint)
	grpcServer.Serve(grpcL)
}

// implemented Tunnel_ConnectServer
type server struct {
	inputStream    chan []byte
	responseStream chan []byte
}

func (s *server) Flow(stream pb.Tunnel_FlowServer) error {

	for {
		select {
		case data := <-s.inputStream:
			log.Printf("Received http request: %v", data)
			err := stream.Send(&pb.Send{})
			if err != nil {
				return err
			}
		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			// keepalive case
			if msg == nil {
				stream.Send(&pb.Send{})
				continue
			}
			log.Printf("Received: %v", msg)
			s.responseStream <- msg.Data
		}
	}
}

func (s *server) Push(req *pb.HTTPRequest, stream pb.Tunnel_PushServer) error {
	log.Printf("Received: %v", req)

	var resp []byte
	data, _ := json.Marshal(req)
	s.inputStream <- data
	for {
		select {
		case <-stream.Context().Done():
			return fmt.Errorf("stream cancelled")
		case data := <-s.responseStream:
			log.Printf("Received: %v", data)
			stream.Send(&pb.HTTPResponse{Id: req.Id, StatusCode: 200, Result: resp})
		}
	}
}
