FROM golang:alpine AS build-env

COPY /location-kinetic.go .

RUN apk update && apk add git ca-certificates
RUN go get golang.org/x/net/websocket && \
    go get golang.org/x/net/proxy && \
    go get github.com/eclipse/paho.mqtt.golang && \
    go get gopkg.in/ini.v1

RUN go build -o location-kinetic location-kinetic.go && ls && pwd


FROM alpine

# RUN apk update && apk add ca-certificates mosquitto-clients
RUN apk update && apk add ca-certificates

WORKDIR /

COPY --from=build-env /go/location-kinetic .

RUN chmod +x location-kinetic

COPY /package_config.ini ./package_config.ini

CMD  ["/location-kinetic"]