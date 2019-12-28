FROM ubuntu:bionic
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apt-get -qq update
RUN apt-get install -qq -y libsass0 ca-certificates curl wget iputils-ping net-tools dnsutils
RUN apt-get clean all
RUN mkdir /config
RUN echo 'bindingPrefix: "http://localhost:28419"' > /config/diecast.yml
COPY bin/diecast-linux-amd64 /usr/bin/diecast

EXPOSE 28419
CMD ["/usr/bin/diecast", "-a", ":28419", "-c", "/config/diecast.yml", "/webroot"]