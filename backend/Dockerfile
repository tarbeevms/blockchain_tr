FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/voting-backend .

FROM alpine:3.21

WORKDIR /app

COPY --from=build /app/voting-backend /app/voting-backend
COPY config.json /app/config.json

EXPOSE 8080

CMD ["/app/voting-backend"]
