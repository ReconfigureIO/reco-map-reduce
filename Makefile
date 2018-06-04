NAME := reco-map-reduce
DESC := Prototype map reduce system on FPGAs
PREFIX ?= usr/local
VERSION := $(shell git describe --tags --always --dirty)
GOVERSION := $(shell go version)
BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILDDATE := $(shell date -u +"%B %d, %Y")
BUILDER := $(shell echo "`git config user.name` <`git config user.email`>")
PKG_RELEASE ?= 1
PKG_NAME = "github.com/ReconfigureIO/$(NAME)"
PROJECT_URL := "https://github.com/ReconfigureIO/$(NAME)"
LDFLAGS := -X 'main.version=$(VERSION)' \
           -X 'main.buildTime=$(BUILDTIME)' \
           -X 'main.builder=$(BUILDER)' \
           -X 'main.goversion=$(GOVERSION)'

.PHONY: test all clean dependencies integration

all: dist/generate-framework

print-% : ; @echo $($*)

test: | dependencies
	go test -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

integration: dist/generate-framework
	find examples -mindepth 1 -maxdepth 1 -type d  -not -path '*/\.*' -exec make -C {} all \;

clean:
	find examples -mindepth 1 -maxdepth 1 -type d  -not -path '*/\.*' -exec make -C {} clean \;
	rm -rf dist

dependencies:
	glide install
	find examples -mindepth 1 -maxdepth 1 -type d  -not -path '*/\.*' -exec make -C {} dependencies \;

dist:
	mkdir -p dist

dist/generate-framework: cmd/generate-framework/main.go dependencies | dist
	go build -o $@ $(PKG_NAME)/cmd/generate-framework
