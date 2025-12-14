FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY ./go.mod ./go.sum ./
WORKDIR /src
RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/app /app/app

COPY --from=builder /src/internal/migrations /app/internal/migrations

EXPOSE 8081

ENTRYPOINT ["/app/app"]
