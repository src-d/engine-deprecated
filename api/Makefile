proto:
	docker run --rm -v $(shell pwd):/data -w /data znly/protoc --go_out=plugins=grpc:. -I=. *.proto