FROM golang:1.26-alpine AS build

WORKDIR /src
ENV CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/mypast ./cmd/mypast

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mypast /mypast

USER nonroot
ENTRYPOINT ["/mypast"]
