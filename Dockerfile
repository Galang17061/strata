FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/strata ./cmd/strata

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && addgroup -S strata && adduser -S strata -G strata
WORKDIR /app
COPY --from=build /out/strata /usr/local/bin/strata
COPY migrations ./migrations
RUN mkdir -p /app/upload && chown -R strata:strata /app
USER strata
ENV STRATA_PORT=5000 STRATA_UPLOAD_DIR=/app/upload STRATA_MIGRATIONS_DIR=/app/migrations
EXPOSE 5000
VOLUME ["/app/upload"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD wget -qO- http://localhost:5000/health || exit 1
ENTRYPOINT ["strata"]
