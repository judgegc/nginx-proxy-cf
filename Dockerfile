FROM golang:alpine

RUN apk update && \
    apk add git \
    gcc \
    linux-headers \
    make

WORKDIR /go/src/app
COPY cf-service/. .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -v ./...


FROM jwilder/nginx-proxy

COPY --from=0 /go/src/app/app /usr/local/bin/cfservice

RUN echo "cfservice: cfservice" >> Procfile