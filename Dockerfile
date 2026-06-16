FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY tools/go.mod tools/go.sum ./tools/
RUN go mod download
RUN cd tools && go mod download
COPY . .
RUN cd tools && go tool oapi-codegen -config ../api/config.yaml ../api/api.yaml && mv ./api.gen.go ../api
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

FROM alpine:3.20
RUN apk add --no-cache curl
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
