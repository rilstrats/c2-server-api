FROM cimg/go:1.23.6

WORKDIR /api

# dependencies
COPY go.mod .
RUN go mod download

# build
COPY . .
RUN go build -o c2-server-api *.go 

EXPOSE 8080
CMD ["./c2-server-api"]
