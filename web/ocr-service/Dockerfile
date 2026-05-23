# =========================================================
# Stage 1: Build
# =========================================================
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    make \
    protobuf-compiler \
    libtesseract-dev \
    libleptonica-dev \
    pkg-config \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install protobuf generators
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy dependency files first
COPY go.mod go.sum* ./

# Download dependencies early for caching
RUN go mod download

# Copy source
COPY . .

# Generate protobufs
RUN protoc \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/ocr.proto

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /tmp/ocr-service .

# =========================================================
# Stage 2: Runtime dependencies
# =========================================================
FROM debian:bookworm-slim AS runtime-deps

RUN apt-get update && apt-get install -y \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-ron \
    ca-certificates \
    libtesseract-dev \
    libleptonica-dev \
    && rm -rf /var/lib/apt/lists/*

# =========================================================
# Stage 3: Distroless final image
# =========================================================
FROM gcr.io/distroless/cc-debian12:nonroot

# Copy shared libraries and runtime binaries
COPY --from=runtime-deps /usr/lib /usr/lib
COPY --from=runtime-deps /lib /lib
COPY --from=runtime-deps /usr/bin/tesseract /usr/bin/tesseract
COPY --from=runtime-deps /usr/share/tesseract-ocr /usr/share/tesseract-ocr
COPY --from=runtime-deps /etc/ssl/certs /etc/ssl/certs

# Copy application
COPY --from=builder /tmp/ocr-service /app/ocr-service

# Tesseract environment
ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/5/tessdata

# Non-root enforced by distroless
USER nonroot:nonroot

EXPOSE 50051

ENTRYPOINT ["/app/ocr-service"]