FROM golang:alpine as builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.* ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/bahn-proxy /bahn-proxy
COPY assets /assets
ENTRYPOINT ["/bahn-proxy"]
