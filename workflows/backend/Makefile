proto:
	go generate generate_proto.go

clean:
	rm -rf goproto/

build: proto
	go build -o home-tasker .

