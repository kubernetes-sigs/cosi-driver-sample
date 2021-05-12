# syntax = docker/dockerfile:1.2.1

FROM --platform=${BUILDPLATFORM} docker.io/golang:1.16.4-buster AS base
ARG BUILDPLATFORM
WORKDIR /usr/src/cosi-driver-sample
COPY go.mod go.sum ./
RUN go mod download

FROM base AS build
ARG TARGETOS
ARG TARGETARCH
RUN --mount=target=. \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
        -mod=readonly \
        -ldflags '-s -w -extldflags -static' \
        -o /out/cosi-driver-sample \
	./cmd/cosi-driver-sample/

FROM base AS test
ARG BUILDOS
ARG BUILDARCH
RUN --mount=target=. \
    CGO_ENABLED=0 \
    GOOS=${BUILDOS} \
    GOARCH=${BUILDARCH} \
    go build \
        -mod=readonly \
        -ldflags '-s -w -extldflags -static' \
	-o /out/cosi-driver-sample \
	./cmd/cosi-driver-sample \
	&& \
    rm -f /out/cosi-driver-sample
ENTRYPOINT ["go", "test", "-mod=readonly", "-v", "./..."]

# gcr.io/distroless/static:nonroot
FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static@sha256:cd784033c94dd30546456f35de8e128390ae15c48cbee5eb7e3306857ec17631 as bin
ARG TARGETPLATFORM

LABEL org.opencontainers.image.authors="Nicolas Trangez <nicolas.trangez@scality.com>" \
      org.opencontainers.image.url="https://github.com/scality/cosi-driver-sample" \
      org.label-schema.url="https://github.com/scality/cosi-driver-sample" \
      org.opencontainers.image.documentation="https://github.com/scality/cosi-driver-sample/blob/hack/README.md" \
      org.label-schema.usage="https://github.com/scality/cosi-driver-sample/blob/hack/README.md" \
      org.opencontainers.image.source="https://github.com/scality/cosi-driver-sample.git" \
      org.label-schema.vcs-url="https://github.com/scality/cosi-driver-sample.git" \
      org.opencontainers.image.vendor="Scality" \
      org.label-schema.vendor="Scality" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.title="cosi-driver-sample" \
      org.label-schema.name="cosi-driver-sample" \
      org.opencontainers.image.description="A sample driver for COSI, the Container Object Storage Interface" \
      org.label-schema.description="A sample driver for COSI, the Container Object Storage Interface" \
      org.label-schema.schema-version="1.0"

COPY --from=build /out/cosi-driver-sample /
ENTRYPOINT ["/cosi-driver-sample"]
