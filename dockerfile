# Оба stage на golang:1.25-alpine — не тянем отдельно library/alpine (меньше запросов к Docker Hub при сборке).
FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wikilivee ./cmd

FROM golang:1.25-alpine

RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=build /out/wikilivee /app/wikilivee
COPY internal/database/migrations /app/internal/database/migrations
COPY outernal /app/outernal

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENV LISTEN_ADDR=:8080

ENTRYPOINT ["/app/wikilivee"]
