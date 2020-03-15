# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')

all:
	go test
	-@go fmt ||exit
