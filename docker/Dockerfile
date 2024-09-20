# Build stage
FROM m.daocloud.io/docker.io/golang:1.23 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum apis ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM docker.m.daocloud.io/alpine:3.15

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Copy profiles directory
COPY istio/profiles ./istio/profiles

# Command to run the executable
CMD ["./main"]
