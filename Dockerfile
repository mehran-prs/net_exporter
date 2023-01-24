FROM golang:1.18.4-alpine as build

MAINTAINER Mehran Prs <mehran@kamva.ir>

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/prometheus/common/version.Version=`cat version`" -o net_exporter ./net_exporter.go

FROM golang:1.18.4-alpine

#RUN apk add ca-certificates

WORKDIR /app

COPY --from=build /app/net_exporter /bin/net_exporter
EXPOSE 9200

ENTRYPOINT ["/bin/net_exporter"]