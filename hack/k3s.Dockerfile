ARG K3S_TAG="v1.30.2-k3s1"
FROM rancher/k3s:$K3S_TAG as k3s
FROM alpine:3
RUN apk add --no-cache util-linux udev lvm2 gptfdisk sgdisk
COPY --from=k3s / /
RUN mkdir -p /etc && \
    echo 'hosts: files dns' > /etc/nsswitch.conf && \
    echo "PRETTY_NAME=\"K3s ${version}\"" > /etc/os-release && \
    chmod 1777 /tmp
VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log
ENV CRI_CONFIG_FILE="/var/lib/rancher/k3s/agent/etc/crictl.yaml"
ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]