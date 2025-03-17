FROM golang:1.23.6-alpine AS build
# setup
WORKDIR /build
# dependencies
COPY go.mod go.sum .
RUN go mod download
# build
COPY . .
RUN go build -o c2-server-api main.go db.go api.go env.go
# alpine uses musl instead of glibc
# CGO_ENABLED=0 required to build statically

FROM alpine:3 AS api
# copy binary
COPY --from=build /build/c2-server-api /usr/local/bin
# run
EXPOSE 8080
CMD ["c2-server-api"]
