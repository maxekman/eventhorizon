all: protoc

protoc:
	protoc -I todo/ -I ../../proto/ --go_out=plugins=grpc,Muuid.proto=github.com/looplab/eventhorizon:todo todo/commands.proto

gogo:
	protoc \
		-I=todo/ \
		-I=../../proto/ \
		-I=$(GOPATH)/src \
		-I=$(GOPATH)/src/github.com/gogo/protobuf/protobuf \
		--gogo_out=plugins=grpc,Meh.proto=github.com/looplab/eventhorizon:todo todo/commands.proto
