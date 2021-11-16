FROM gcr.io/distroless/static:latest
LABEL maintainers="Kubernetes COSI Authors"
LABEL description="Object Storage Sidecar"

COPY ./bin/sample-cosi-driver sample-cosi-driver
ENTRYPOINT ["/sample-cosi-driver"]
