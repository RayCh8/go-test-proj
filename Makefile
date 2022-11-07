PATH := ${CURDIR}/bin:$(PATH)

goexe = $(shell go env GOEXE)

# make
# rebuild all dependencies and run all code generators, use it as an initial command
default: cleanup codegen

# make init
.PHONY: init
init:
	cp .env.example .env;

# make codegen
# ensure protobuf set up and run all code generators
.PHONY: codegen
codegen: protoc.setup$(go_exe) protoc.codegen go.setup$(go_exe) go.mocks go.gen go.fmt

# make cleanup
# remove protobuf installs and generated golang binary.
.PHONY: cleanup
cleanup: protoc.cleanup go.cleanup

# make test
# run tests without starting related service (eg. DB)
.PHONY: test
test: go.test go.testcoverage

# make test-report
# get test coverage report
.PHONY: testcoverage
testcoverage:
	go tool cover -html=coverage.out

# make build
# build golang binary under cmd/ and copy STATIC_FILES as configured
.PHONY: build
build: go.build

# make ci-build
# a single command to run test and build golang binary for CI
# in local development environment use docker.ci-build instead
.PHONY: ci-build
ci-build: init test build

# make rpc
# start
.PHONY: rpc
rpc:
	"Starting RPC always need Docker. Redirect to `make docker.rpc`";
	make docker.rpc;

# Sugar commands,
# same as previous commands but base on docker engine
# = = = = = = = = = = = = = = = = = = = = = = = = = = = =
docker.codegen:
	sh dockerbuild.sh codegen;

docker.test:
	sh dockerbuild.sh test;

docker.rpc:
	sh dockerbuild.sh rpc;

docker.build:
	sh dockerbuild.sh build;

docker.ci-build:
	sh dockerbuild.sh ci-build;

# = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =
# = Makefile internal commands, only use these commands if it's necessary =
# = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =

# Golang build commands
# = = = = = = = = = = = = = = = = = = = = = = = = = = = =
go.build:
	ls cmd | xargs -I {} bash -c 'go build -o bin/cmd/{} cmd/{}/*.go';
	for file in ${STATIC_FILES}; do cp -R $$file bin/cmd/; done;
	for file in ${SCRIPT_FILES}; do cp -R $$file bin/cmd/; done;

go.mocks: go.setup
	mockery --all --keeptree --output=./internal

go.gen: go.setup
# for example: abice/go-enum
	go generate ./...

go.cleanup:
	rm -rf bin/cmd;

go.test:
	go test -v -coverprofile=coverage.out ./...

go.testcoverage:
	sh ./testcoverage.sh

go.fmt:
	go fmt ./pkg/pb/*.go;
	goimports --local github.com/AmazingTalker -w ./pkg/pb/*.go;

go.setup: bin/mockgen$(go_exe) bin/goimports$(go_exe) bin/mockery$(go_exe)

bin/mockgen$(go_exe): go.mod
	go get github.com/golang/mock/mockgen;
	go build -o $@ github.com/golang/mock/mockgen
bin/goimports$(go_exe): go.mod
# FIXME: It's weird that goimports depending on mockgen. Change the order leading the error.
	go build -o $@ golang.org/x/tools/cmd/goimports
bin/mockery$(go_exe): go.mod
# DEPRECATED soon. It's better to install it by yourself
# How to install: https://github.com/vektra/mockery#installation
	go get github.com/vektra/mockery/v2
	go build -o $@ github.com/vektra/mockery/v2

# Protobuf, protoc commands
# = = = = = = = = = = = = = = = = = = = = = = = = = = = =
protoc.codegen:
	protoc \
		-I=. \
		-I=third_party \
		--svc_out=. \
		--gogoslick_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,plugins=grpc:. \
		--gogoslick_opt=paths=source_relative \
		./pkg/pb/rpc.proto;

.PHONY: protoc.cleanup
protoc.cleanup:
	rm -rf bin;
	rm -rf third_party;

protoc.setup$(go_exe): bin/protoc bin/protoc-gen-go-grpc bin/gogoproto-gogoproto$(go_exe) bin/protoc-gen-gogo$(go_exe) bin/protoc-gen-gogoslick$(go_exe) bin/gogoproto-jsonpb$(go_exe) bin/protoc-gen-svc$(go_exe)
	mkdir -p third_party; \
	chmod -R 755 $(shell go list -f '{{ .Dir }}' github.com/AmazingTalker/protoc-gen-svc)/third_party/* third_party/; \
	cp -r $(shell go list -f '{{ .Dir }}' github.com/AmazingTalker/protoc-gen-svc)/third_party/* third_party/;

# base protoc, grpc
bin/protoc: go.mod
	go mod download google.golang.org/grpc/cmd/protoc-gen-go-grpc;
	go install google.golang.org/protobuf/cmd/protoc-gen-go
bin/protoc-gen-go-grpc: go.mod
	go build -o $@ google.golang.org/grpc/cmd/protoc-gen-go-grpc
# protoc plugins - gogoprotobuf
bin/gogoproto-gogoproto$(go_exe): go.mod
	go build -o $@ github.com/gogo/protobuf/gogoproto
bin/protoc-gen-gogo$(go_exe): go.mod
	go build -o $@ github.com/gogo/protobuf/protoc-gen-gogo
bin/protoc-gen-gogoslick$(go_exe): go.mod
	go build -o $@ github.com/gogo/protobuf/protoc-gen-gogoslick
bin/gogoproto-jsonpb$(go_exe): go.mod
	go build -o $@ github.com/gogo/protobuf/jsonpb
# AmazingTalker service template engine
bin/protoc-gen-svc$(go_exe): go.mod
	go build -o $@ github.com/AmazingTalker/protoc-gen-svc


