FROM golang:1.19-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o contributord ./cmd/contributord

FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /app/contributord /app/contributord
COPY --from=build /app/static /app/static
COPY --from=build /app/templates /app/templates

EXPOSE 6000

USER nonroot:nonroot

ENTRYPOINT ["/app/contributord"]
