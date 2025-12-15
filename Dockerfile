# ---------- Build stage ----------
FROM golang:1.24-alpine AS builder

# Install git (required for go mod in some cases)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o server cmd/server/main.go


# ---------- Runtime stage ----------
FROM alpine:3.20

# Create non-root user (security best practice)
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Use non-root user
USER app

# Expose the port your app listens on
EXPOSE 5000

# Run the binary
CMD ["./server"]
