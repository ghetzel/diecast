FROM golang:1.14.4-alpine3.12
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apk update && apk add --no-cache bash gcc g++ libsass-dev libsass ca-certificates curl wget make socat git jq
RUN GO111MODULE=on go get github.com/ghetzel/diecast/cmd/diecast
RUN rm -rf /go/pkg /go/src
ADD https://storage.googleapis.com/kubernetes-release/release/v1.18.3/bin/linux/amd64/kubectl /usr/bin/kubectl
RUN chmod -v 0755 /usr/bin/kubectl
RUN mkdir /config /webroot
RUN echo 'bindingPrefix: "http://localhost:28419"' > /config/diecast.yml

EXPOSE 28419
ENV DIECAST_ALLOW_ROOT_ACTIONS true
WORKDIR /webroot
CMD ["/usr/bin/diecast", "-a", ":28419", "-c", "/config/diecast.yml", "/webroot"]