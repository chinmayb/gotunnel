package http

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	pb "github.com/chinmayb/gotunnel/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type proxy struct {
	tunnelClient pb.TunnelClient
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(w, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}
	// rpc calls
	result, err := p.tunnelClient.Push(req.Context(), &pb.HTTPRequest{
		Method: req.Method,
		Url:    req.URL.String(),
		// Header: req.Header,
		// Body:   req.Body,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var results []byte
	for {
		receive, err := result.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.WriteHeader(int(receive.StatusCode))
		results = append(results, receive.GetResult()...)
	}
	io.Copy(w, strings.NewReader(string(results)))
}

func getClient(grpcEndpoint string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(grpcEndpoint)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", grpcEndpoint, cerr)
			}
			return
		}
	}()
	return conn, nil
}

func proxyHandler() {
	var addr = flag.String("addr", "127.0.0.1:8080", "The addr of the application.")
	flag.Parse()
	client, err := getClient("localhost:50051")
	if err != nil {
		log.Fatal(err)
	}

	handler := &proxy{tunnelClient: pb.NewTunnelClient(client)}

	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
