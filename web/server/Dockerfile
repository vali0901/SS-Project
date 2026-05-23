# Stage 1: Build environment
FROM golang:1.25-alpine AS builder

# 1. Install build-time dependencies (compilers and dev headers)
# These change rarely and should be at the top
RUN apk add --no-cache \
    tesseract-ocr-dev \
    leptonica-dev \
    g++ \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# 2. Cache Go modules
# Copy ONLY the dependency files first
COPY go.mod go.sum ./
RUN go mod download

# 3. Copy source code and build
# This layer will be the only one invalidated when you change your code
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o mqtt ./main.go

# Stage 2: Final runtime image
FROM alpine@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

# 4. Install ONLY runtime dependencies
# We remove -dev packages and compilers to keep the image small
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    tesseract-ocr-data-ron \
    ffmpeg \
    libstdc++ \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# 5. Copy the binary from the builder stage
COPY --from=builder /app/mqtt /app/

EXPOSE 8080

CMD ["./mqtt"]
