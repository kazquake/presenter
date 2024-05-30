# Build stage
FROM golang:1.22.0 AS build-env
ENV GO111MODULE=on
ENV CGO_ENABLED=0

WORKDIR /app/presenter
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN go build -o presenter
RUN chmod +x presenter

# Release stage
FROM alpine:latest AS release
WORKDIR /app/
COPY --from=build-env /app/presenter/presenter .
ENV WORKDIR "/app/"
ENV PATH "${WORKDIR}:${PATH}"

EXPOSE 8080/tcp 8080/udp

CMD ["./presenter"]