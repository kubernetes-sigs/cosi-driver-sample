#
# BUILDER
#
FROM docker.io/library/golang:1.24.2 AS builder

WORKDIR /buildroot

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the source.
COPY pkg/ pkg/
COPY cmd/ cmd/

# Disable CGO.
ENV CGO_ENABLED=0

# Build the image.
RUN go build -trimpath -o build/sample-cosi-driver cmd/sample-cosi-driver/*.go

#
# FINAL IMAGE
#
FROM gcr.io/distroless/static:latest AS runtime

LABEL org.opencontainers.image.maintainers="Kubernetes Authors"
LABEL org.opencontainers.image.description="Container Object Storage Interface (COSI) Sample Driver"
LABEL org.opencontainers.image.title="COSI Sample Driver"
LABEL org.opencontainers.image.source="https://github.com/kubernetes-sigs/cosi-driver-sample"
LABEL org.opencontainers.image.licenses="APACHE-2.0"

COPY --from=builder /buildroot/build/sample-cosi-driver /sample-cosi-driver

ENTRYPOINT [ "/sample-cosi-driver" ]
