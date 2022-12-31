FROM golang:1.19-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o contributed-to

FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /app/contributed-to /contributed-to

EXPOSE 6000

USER nonroot:nonroot

ENTRYPOINT ["/contributed-to"]
