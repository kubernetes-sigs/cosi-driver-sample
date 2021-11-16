FROM gcr.io/distroless/static:latest
LABEL maintainers="DELL EMC ObjectScale"
LABEL description="ObjectScale COSI Provisioner"

COPY ./bin/sample-cosi-driver sample-cosi-driver
ENTRYPOINT ["/sample-cosi-driver"]