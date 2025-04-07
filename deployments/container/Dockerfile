# Use a minimal Go image
FROM golang:1.24 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go source code into the container
COPY . .

# Download dependencies
RUN go mod tidy && go mod vendor

# Build the driver binary
RUN CGO_ENABLED=0 go build -o dra-cpu-kubeletplugin ./cmd/dra-cpu-kubeletplugin

# Use a lightweight base image
FROM alpine:latest

# Copy the compiled binary from the builder stage
COPY --from=builder /app/dra-cpu-kubeletplugin /bin/dra-cpu-kubeletplugin

# Run dra-cpu-kubeletplugin
ENTRYPOINT ["/bin/dra-cpu-kubeletplugin"]