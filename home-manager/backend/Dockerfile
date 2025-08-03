FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
COPY templates/ /app/templates/
RUN go build -o hometasker

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/hometasker /app/hometasker
COPY --from=builder /app/templates /app/templates/
CMD ["./hometasker"]

