FROM ubuntu:eoan
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apt-get -qq update && apt-get install -qq -y libsass1 ca-certificates curl wget iputils-ping net-tools dnsutils make socat bzr git
RUN apt-get clean all
ADD https://storage.googleapis.com/kubernetes-release/release/v1.18.3/bin/linux/amd64/kubectl /usr/bin/kubectl
RUN chmod -v 0755 /usr/bin/kubectl
RUN mkdir /config
RUN echo 'bindingPrefix: "http://localhost:28419"' > /config/diecast.yml
COPY bin/diecast-linux-amd64 /usr/bin/diecast

EXPOSE 28419
ENV DIECAST_ALLOW_ROOT_ACTIONS true
CMD ["/usr/bin/diecast", "-a", ":28419", "-c", "/config/diecast.yml", "/webroot"]
