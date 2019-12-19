VERSION=0.0.3
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
GO111MODULE=on

all: myps

.PHONY: myps

myps: main.go
	go build $(LDFLAGS) -o myps

linux: main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o myps

deps:
	go get -d
	go mod tidy

deps-update:
	go get -u -d
	go mod tidy

clean:
	rm -rf myps

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin master
	goreleaser --rm-dist
