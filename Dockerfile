# Use the official Golang image for building the application
FROM golang:1.22 as builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./
COPY go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application
RUN go build -o /pyproxy ./cmd/proxy

# Use a minimal image to run the application
FROM debian:buster-slim

COPY --from=builder /pgproxy /pgproxy

# Expose the application on port 5432
EXPOSE 5432

# Run the executable
ENTRYPOINT ["/pgproxy"]
