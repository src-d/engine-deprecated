# published as srcd/cli-daemon

FROM golang:1.10 as builder

ENV ROOTPATH=github.com/src-d/engine
ARG go_build_tags
ENV BUILD_TAGS=${go_build_tags}

# update gRPC api?
# RUN go get github.com/golang/protobuf/protoc-gen-go

ADD . /go/src/${ROOTPATH}
WORKDIR /go/src/${ROOTPATH}
RUN TAG=$(git describe --tags) && go install -tags "${BUILD_TAGS}" -ldflags "-X main.version=${TAG}" "${ROOTPATH}/cmd/srcd-server"

FROM alpine
RUN apk update && \
    apk add ca-certificates libstdc++ libxml2-dev && \
    rm -rf /var/cache/apk/*
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=builder /go/bin/srcd-server /
ENTRYPOINT ["/srcd-server"]
