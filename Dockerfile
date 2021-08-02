FROM golang:alpine as build

RUN apk add gcc musl-dev

COPY ./ /src
RUN \
	cd /src && \
	go build -o /blog-log-agent main.go


FROM alpine:3.13
COPY --from=build /blog-log-agent /

ENTRYPOINT ["/blog-log-agent"]
