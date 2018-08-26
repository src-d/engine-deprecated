# published as srcd-cli/daemon

FROM golang:1.10 as builder

RUN apt-get update && apt-get install -y git libxml2-dev
ENV ROOTPATH=github.com/src-d/engine-cli

# update gRPC api?
# RUN go get github.com/golang/protobuf/protoc-gen-go

ADD . /go/src/${ROOTPATH}
WORKDIR /go/src/${ROOTPATH}
RUN HEAD=$(git rev-parse HEAD) && go install -ldflags "-X main.version=${HEAD}" "${ROOTPATH}/cmd/srcd-server"

# why is it that libuast makes it so hard?

# FROM debian:jessie
# RUN apt-get update && apt-get install -y libxml2-dev
# COPY --from=builder /go/bin/srcd-server /
# ENTRYPOINT ["/srcd-server"]
CMD /go/bin/srcd-server --workdir="${WORKDIR}"