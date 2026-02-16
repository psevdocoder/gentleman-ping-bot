FROM golang:1.25.3-alpine AS builder

WORKDIR /build

RUN apk add --no-cache ca-certificates git

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN go build -o app ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/app .

VOLUME ["./values"]

CMD ["./app"]