FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o forming-app .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/forming-app .

# Copy views directory (required for templates)
COPY --from=builder /app/views ./views

# Copy public directory (required for static assets)
COPY --from=builder /app/public ./public

# Expose port
EXPOSE 3000

# Run the application
CMD ["./forming-app"]
