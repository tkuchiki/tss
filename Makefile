VERSION := commit-$(shell git rev-parse --short HEAD)

build:
	go build -trimpath -ldflags '-X main.version=$(VERSION)'
