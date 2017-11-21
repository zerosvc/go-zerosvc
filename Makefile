# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')

all: vendor glide.lock
	go test
	-@go fmt ||exit
clean:
	rm -rf vendor
vendor: glide.lock
	glide install && touch vendor
glide.lock: glide.yaml
	glide update && touch glide.lock
glide.yaml:
