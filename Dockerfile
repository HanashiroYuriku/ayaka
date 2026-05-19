# ==========================================
# STAGE 1: BUILD STAGE
# ==========================================
# Use an official Golang image as the build environment
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./

# Download all required dependencies (including Cobra and Viper)
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Golang application into a single binary named 'furina_app'
RUN go build -o furina_app main.go

# ==========================================
# STAGE 2: RUN STAGE
# ==========================================
# Use a fresh, lightweight Alpine image for the final runtime
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/furina_app .

# Copy configuration and environment files
COPY config.yaml .
COPY .env .

# Inform Docker that the application listens on port 8000
EXPOSE 8000

# START THE APPLICATION!
# Note: This is equivalent to running 'go run main.go svc' in your terminal
CMD ["./furina_app", "svc"]