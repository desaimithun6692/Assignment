FROM  golang:1.25-alpine AS builder
WORKDIR /app
# COPY go.mod go.sum ./
COPY . .
RUN go mod tidy
RUN go mod download
# Build the function runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./main.go
RUN chmod +x main

# Run stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
# Cloud Functions environment variable to target the specific function
ENV FUNCTION_TARGET=ProcessMessage
ENV PORT=8080
EXPOSE 8080
CMD ["./main"]