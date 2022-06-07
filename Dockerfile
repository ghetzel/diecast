FROM golang:1.18.3-alpine3.16
MAINTAINER Gary Hetzel <its@gary.cool>

ENV GO111MODULE on
RUN apk update && apk add --no-cache bash gcc g++ libsass-dev libsass ca-certificates curl wget make socat git jq
RUN go install github.com/ghetzel/diecast/cmd/diecast@v1.21.2
RUN rm -rf /go/pkg /go/src
RUN mv /go/bin/diecast /usr/bin/diecast
ADD https://storage.googleapis.com/kubernetes-release/release/v1.24.1/bin/linux/amd64/kubectl /usr/bin/kubectl
RUN chmod -v 0755 /usr/bin/kubectl
RUN mkdir /config /webroot
RUN echo 'bindingPrefix: "http://localhost:28419"' > /config/diecast.yml
RUN rm -rf /usr/local/go /usr/libexec/gcc

RUN test -x /usr/bin/diecast
RUN test -x /usr/bin/kubectl

EXPOSE 28419
ENV DIECAST_ALLOW_ROOT_ACTIONS true
WORKDIR /webroot
ENTRYPOINT ["/usr/bin/diecast"]
CMD ["-a", ":28419", "-c", "/config/diecast.yml", "/webroot"]
