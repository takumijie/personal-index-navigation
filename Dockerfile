# syntax=docker/dockerfile:1

FROM node:24-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.27-alpine AS backend-build
WORKDIR /app
ARG GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY web ./web
COPY --from=frontend-build /app/frontend/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/homepage ./cmd/server \
    && mkdir -p /out/data /out/storage/music /out/storage/images \
    && chown -R 65532:65532 /out/data /out/storage

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=backend-build /out/homepage /usr/local/bin/homepage
COPY --from=backend-build --chown=65532:65532 /out/data /data
COPY --from=backend-build --chown=65532:65532 /out/storage /storage
ENV HOMEPAGE_SERVER_MODE=release \
    HOMEPAGE_SERVER_LISTEN=0.0.0.0 \
    HOMEPAGE_DATABASE_PATH=/data/homepage.db \
    HOMEPAGE_STORAGE_PATH=/storage
EXPOSE 8080
VOLUME ["/data", "/storage"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/homepage", "--healthcheck"]
ENTRYPOINT ["/usr/local/bin/homepage"]
