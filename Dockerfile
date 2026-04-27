# syntax=docker/dockerfile:1.7

FROM golang:1.26.2 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api   ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fetch ./cmd/fetch

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api   /usr/local/bin/api
COPY --from=build /out/fetch /usr/local/bin/fetch
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
