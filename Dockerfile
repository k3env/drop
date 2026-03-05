FROM scratch

WORKDIR /app
COPY drop drop
ENTRYPOINT ["/app/drop"]