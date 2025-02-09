# Create user
FROM alpine:latest AS builder

RUN adduser --disabled-password --uid=1001 appuser

# Run the binary
FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
USER appuser

COPY ./bin/user-profile-service /app/bin/service

EXPOSE 8080

ENTRYPOINT ["/app/bin/service"]