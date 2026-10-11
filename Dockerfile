FROM golang:1.25.4-alpine3.21 AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
 && addgroup -g 1000 -S webhook \
 && adduser -u 1000 -S webhook -G webhook

COPY --from=build /workspace/webhook /usr/local/bin/webhook

USER 1000

ENTRYPOINT ["webhook"]
