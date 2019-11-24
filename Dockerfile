FROM ubuntu:bionic
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apt-get -qq update
RUN apt-get install -y libsass0 ca-certificates
RUN apt-get clean all
COPY bin/diecast-linux-amd64 /usr/bin/diecast

EXPOSE 28419
CMD ["/usr/bin/diecast", "-a", ":28419", "/webroot"]
