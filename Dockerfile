FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/recollect ./cmd/recollect

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 65532 app
COPY --from=build /out/recollect /usr/local/bin/recollect
USER app
EXPOSE 8080
ENTRYPOINT ["recollect"]
