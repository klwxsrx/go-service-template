# Create user
FROM alpine:latest AS builder

RUN adduser --disabled-password --uid=1001 appuser

# Run the binary
FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
USER appuser

COPY ./bin/idk-cleaner-task /app/bin/task

ENTRYPOINT ["/app/bin/task"]