FROM golang:1.14-alpine as builder

WORKDIR /go/src/app
COPY . /go/src/app

RUN CGO_ENABLED=0 go build -v -ldflags="-s -w"

FROM gcr.io/distroless/static

COPY --from=builder /go/src/app/stremio-top-movies /

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/stremio-top-movies"]
CMD ["-bindAddr", "0.0.0.0", "-dataDir", "/data"]
