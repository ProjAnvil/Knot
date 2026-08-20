# syntax=docker/dockerfile:1

# Stage 1: build frontend
FROM oven/bun:1 AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

# Stage 2: build backend with embedded frontend
FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Overwrite any stale prebuilt assets with the fresh frontend build
RUN rm -rf internal/embedded/frontend_dist
COPY --from=frontend /app/frontend/dist ./internal/embedded/frontend_dist
RUN CGO_ENABLED=0 go build -tags embedfrontend -ldflags="-s -w" -o /knot-server ./cmd/server

# Stage 3: runtime
FROM alpine:3
RUN adduser -D -h /home/knot knot && mkdir -p /home/knot/.knot && chown -R knot:knot /home/knot
COPY --from=backend /knot-server /knot-server
USER knot
EXPOSE 3000
ENTRYPOINT ["/knot-server"]
