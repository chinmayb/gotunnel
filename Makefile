proto:
	docker run --rm -u `id -u`:`id -g` -e GOCACHE=/go -e CGO_ENABLED=0 -v /Users/cbharadwaj/go/src/github.com/chinmayb/gotunnel:/go/src/github.com/chinmayb/gotunnel infoblox/atlas-gentool:v21.3 --go_out=plugins=grpc:. --grpc-gateway_out=logtostderr=true,allow_delete_body=true:.  github.com/chinmayb/gotunnel/pkg/pb/tunnel.proto

