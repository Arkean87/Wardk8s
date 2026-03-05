FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wardk8s ./cmd/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
RUN addgroup -S wardk8s && adduser -S wardk8s -G wardk8s
COPY --from=builder /app/wardk8s /usr/local/bin/wardk8s
USER wardk8s
ENTRYPOINT ["wardk8s"]
