# syntax=docker/dockerfile:1

FROM oven/bun:alpine AS bun-runtime

FROM golang:1.25-alpine AS builder
ARG VERSION
WORKDIR /src
RUN apk add --no-cache git make libstdc++ libgcc
COPY --from=bun-runtime /usr/local/bin/bun /usr/local/bin/bun
COPY --from=bun-runtime /usr/local/bin/bunx /usr/local/bin/bunx
ENV PATH="/usr/local/bin:${PATH}"
COPY Makefile VERSION ./
COPY frontend/ ./frontend/
COPY backend-go/ ./backend-go/
RUN cd frontend && bun install --frozen-lockfile
RUN cd backend-go && go mod download
RUN if [ -n "${VERSION}" ]; then VERSION=${VERSION} make build; else make build; fi

FROM alpine:3.21 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata curl && \
    mkdir -p /app/data
COPY --from=builder /src/dist/newapi-tools /app/newapi-tools
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8000/api/health || exit 1
CMD ["/app/newapi-tools"]
