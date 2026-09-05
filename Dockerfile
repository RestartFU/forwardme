# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/forwardme \
    ./cmd && \
    mkdir -p /storage/certmagic

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build --chown=nonroot:nonroot /out/forwardme /app/forwardme
COPY --from=build --chown=nonroot:nonroot /storage /var/lib/forwardme

USER nonroot:nonroot

EXPOSE 80 443
VOLUME ["/var/lib/forwardme"]

ENTRYPOINT ["/app/forwardme"]
