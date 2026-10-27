FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /bigtable-emulator .

FROM alpine:3.24.3

COPY --from=build /bigtable-emulator /usr/local/bin/bigtable-emulator

EXPOSE 8086

ENTRYPOINT ["bigtable-emulator"]
