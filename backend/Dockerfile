FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H unesarent

WORKDIR /app
COPY --from=builder /out/server ./server
COPY --from=builder /src/assets ./assets

# Direktori dibuat saat build agar user non-root bisa menulis saat runtime.
RUN mkdir -p data/ktm data/listings data/rental-documents \
    && chown -R unesarent:unesarent /app

USER unesarent

ENV APP_ENV=production
ENV PORT=8080

EXPOSE 8080

CMD ["./server"]
