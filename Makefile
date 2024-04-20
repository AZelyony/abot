APP=$(shell basename $(shell git remote get-url origin))
#REGISTRY=gcr.io/azelyony
#REGISTRY=azelyony
REGISTRY=ghcr.io/abot-16207
VERSION=$(shell git describe --tags --abbrev=0)-$(shell git rev-parse --short HEAD)
TARGETOS=linux #darwin windows
TARGETARCH=amd64 # arm64

#Use "make build TARGETOS=windows TARGETARCH=amd64"

format: 
	gofmt -s -w ./

get:
	go get

lint:
	golint

test: 
	go test -v

build: format get
#	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${shell dpkg --print-architecture} go build -v -o abot -ldflags "-X="github.com/azelyony/abot/cmd.appVersion=${VERSION}
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o abot -ldflags "-X="github.com/azelyony/abot/cmd.appVersion=${VERSION}

image:
#	docker build . -t ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}
	docker build . -t ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}  --build-arg TARGETARCH=${TARGETARCH}

push:
	docker push ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}

clean: 
	rm -rf abot
	#за умовами завдання виконуємо docker rmi <IMAGE_TAG> 
	docker rmi ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}