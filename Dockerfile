FROM golang:1.24-alpine AS builder

WORKDIR /app
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /api-server ./main.go

FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /api-server /api-server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/api-server"]
