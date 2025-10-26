FROM golang:1.23 as build
WORKDIR /app
COPY . .
RUN go mod init cyberpank_neon
RUN go mod tidy
RUN go build -o game-api .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /app/game-api .
COPY templates/ ./templates/
EXPOSE 8080
ENTRYPOINT ["./game-api"]

