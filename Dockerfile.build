FROM golang:1.14.4-alpine3.12
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apk update && apk add --no-cache bash libsass-dev ca-certificates curl wget make socat git jq gcc libc-dev
RUN go get -u github.com/mjibson/esc
RUN go get -u golang.org/x/tools/cmd/goimports
WORKDIR /project

CMD ["make", "deps", "test", "build"]