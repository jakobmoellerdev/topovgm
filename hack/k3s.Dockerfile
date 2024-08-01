ARG K3S_TAG="v1.30.2-k3s1"
FROM rancher/k3s:$K3S_TAG as k3s

FROM ubuntu:latest

RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

RUN apt-get update && \
    apt-get -y install gnupg2 curl lvm2 util-linux

COPY --from=k3s / /

RUN mkdir -p /etc && \
    echo 'hosts: files dns' > /etc/nsswitch.conf

RUN chmod 1777 /tmp

VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log

ENV PATH="$PATH:/bin/aux"

ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]