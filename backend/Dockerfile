FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /bin/api ./cmd/api
RUN go build -o /bin/worker ./cmd/worker

FROM alpine:3.20
WORKDIR /app
COPY --from=build /bin/api /usr/local/bin/api
COPY --from=build /bin/worker /usr/local/bin/worker
RUN mkdir -p /app/data
EXPOSE 8080
CMD ["api"]
