FROM golang:1.19.2-alpine3.16
ENV CGO_ENABLED=0
RUN apk add make
WORKDIR /go/src/github.com/form3interview
COPY . .
ENTRYPOINT ["make", "test"]