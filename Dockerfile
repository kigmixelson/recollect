FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/recollect ./cmd/recollect

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/recollect /usr/local/bin/recollect
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/recollect"]
