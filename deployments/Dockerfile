# Golang apps build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o webapp ./cmd/webapp
RUN go build -o importer ./cmd/importer
RUN go build -o reporter ./cmd/reporter

# Deployment of the web app
FROM alpine:3.20 AS webapp_deploy
WORKDIR /app
COPY --from=builder /app/webapp .
EXPOSE 8080
CMD ["./webapp"]

# deployment of the reporter application
FROM alpine:3.20 AS reporter_deploy
WORKDIR /app
COPY --from=builder /app/reporter .
CMD ["./reporter"]

# image for the importer app (meant for local execution only)
FROM alpine:3.20 AS importer_deploy
WORKDIR /app
COPY --from=builder /app/importer .
CMD ["./importer"]

# optional postgresql image in case a managed database instance is not used in production
FROM postgres:16-alpine AS db_deploy
RUN mkdir /sql-src
COPY tables /sql-src/tables
COPY docker-entrypoint-initdb.d/* /docker-entrypoint-initdb.d


