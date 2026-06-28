# compile
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o ubi-go .

# using upgrade below to ensure the latest security patches
# and vulnerabilities are handled
FROM debian:latest

RUN mkdir -p /app/ubi-go
WORKDIR /app/ubi-go

RUN apt-get update && apt-get upgrade -y && apt-get install -y \
    ca-certificates \
    chromium \
    chromium-sandbox \
    --no-install-recommends \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/ubi-go ./ubi-go

# need both .env and .project-root copied
# otherwise the envparser package cannot load the .env file
# contents and everything will break
COPY .env ./.env
COPY .project-root ./.project-root

RUN chmod +x ubi-go
EXPOSE 8080

ENTRYPOINT ["./ubi-go"]