FROM node:24-bookworm-slim AS frontend
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27-bookworm AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 go build -trimpath -o /clyra ./cmd/clyra

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 clyra && mkdir -p /app/data && chown clyra /app/data
WORKDIR /app
COPY --from=backend /clyra /usr/local/bin/clyra
COPY --from=frontend /src/dist ./dist
ENV CLYRA_ENV=production CLYRA_ADDR=:8080 CLYRA_DATABASE_PATH=/app/data/clyra.db CLYRA_STATIC_DIR=/app/dist
USER clyra
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["clyra"]
