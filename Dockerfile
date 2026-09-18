# --- build stage ---
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cached separately from source so `go mod download` only reruns when
# dependencies actually change, not on every code edit.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 appuser
USER appuser

COPY --from=build /out/server /usr/local/bin/server

EXPOSE 8081
ENTRYPOINT ["server"]
