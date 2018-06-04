FROM ubuntu:artful

MAINTAINER Josh Bohde <josh.bohde@reconfigure.io>

ENV GOPATH /mnt

RUN apt-get update && apt-get install -y \
    make git golang \
    && go get golang.org/x/tools/cmd/bundle \
    && go get github.com/Masterminds/glide

ENV PATH $GOPATH/bin:$PATH
