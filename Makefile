.PHONY: build
build:
	go build -o hraft main.go

run/node1:
	./hraft -bootstrap -id node1 -haddr localhost:21001 -raddr localhost:22001 -data node1

run/node2:
	./hraft -id node2 -join localhost:21001 -haddr localhost:21002 -raddr localhost:22002 -data node2

run/node3:
	./hraft -id node3 -join localhost:21001 -haddr localhost:21003 -raddr localhost:22003 -data node3
