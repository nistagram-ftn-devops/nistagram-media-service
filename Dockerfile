FROM golang:1.16.3-alpine as build-env
EXPOSE 8000

WORKDIR /go/src/media-service

# install dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# copy source code
COPY . .

# Build the binary
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o .
# CMD ["bin/sh", "./nistagram-media-service"]
CMD ["go", "run", "."]
