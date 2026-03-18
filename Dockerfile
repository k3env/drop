FROM alpine:latest

WORKDIR /app
COPY drop drop
RUN mkdir -p /app/tmp
ENTRYPOINT ["/app/drop"]