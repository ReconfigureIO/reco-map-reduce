VERSION := $(shell git describe --tags --always --dirty)
CONFIG := reco.yml

.PHONY: all clean dependencies

all: main.go

clean:
	rm -rf mapreduce.go main.go
	rm -rf vendor

dependencies:
	glide install

mapreduce.go: $(CONFIG)
	generate-framework -config $(CONFIG) -output mapreduce.go

main.go: mapreduce.go | dependencies
	bundle -prefix " " -o main.go .
