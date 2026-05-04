# ============================================================
# TomatoTogether - All-in-one Docker image
# Build: docker build -t tomatogether .
# Run:   docker run -p 8080:8080 -v tomatogether-data:/app/data tomatogether
# ============================================================

# Stage 1: Build frontend (SvelteKit static export)
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend (Go with CGO for SQLite)
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /app/bin/tomatogether main.go

# Stage 3: Minimal runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata sqlite-libs
WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/bin/tomatogether .

# Copy database migrations
COPY --from=backend-builder /app/backend/migrations ./migrations

# Copy frontend static files
COPY --from=frontend-builder /app/frontend/build ./static

# Create data directory for SQLite
RUN mkdir -p /app/data

ENV PORT=8080
ENV STATIC_DIR=/app/static
ENV DB_PATH=/app/data/tomatogether.db

EXPOSE 8080

VOLUME ["/app/data"]

CMD ["./tomatogether"]
