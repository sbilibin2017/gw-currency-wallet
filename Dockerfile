FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git bash
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG BUILD_VERSION="N/A"
ARG BUILD_COMMIT="N/A"
ARG BUILD_DATE="N/A"

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-X 'main.buildVersion=${BUILD_VERSION}' -X 'main.buildCommit=${BUILD_COMMIT}' -X 'main.buildDate=${BUILD_DATE}'" \
    -o gw-wallet cmd/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/gw-wallet .
COPY example.env .

EXPOSE 8080
ENTRYPOINT ["sh", "-c", "./gw-wallet -c example.env"]
