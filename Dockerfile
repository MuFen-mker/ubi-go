FROM debian:latest

RUN mkdir -p /app/ubi-go
WORKDIR /app/ubi-go

RUN apt-get update && apt-get install -y \
    ca-certificates \
    chromium \
    chromium-sandbox \
    --no-install-recommends \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY . /app/ubi-go

RUN chmod +x ubi-go

EXPOSE 8080

ENTRYPOINT ["./ubi-go"]