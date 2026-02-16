# Stage 1: Builder
FROM harbor.eu-west-1.csq.io/docker-hub/library/golang:1.24-alpine AS builder
RUN apk add --no-cache make

WORKDIR /go/src/secretMessage
COPY . .

#build the application
RUN make build

# Stage 2: Runtime (Alpine base)
FROM harbor.eu-west-1.csq.io/docker-hub/library/alpine

# Create user and group
RUN addgroup -g 10001 appuser && \
    adduser -D -u 10001 -G appuser appuser

# Copy the built binary from the builder stage
COPY --from=builder /go/src/secretMessage/secretmessage /go/bin/secretmessage

EXPOSE 8080
USER appuser:appuser
ENTRYPOINT ["/go/bin/secretmessage"]
