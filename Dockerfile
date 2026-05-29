FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.22
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /server /server
COPY database/migrations/ /database/migrations/
EXPOSE 8080
CMD ["/server"]
