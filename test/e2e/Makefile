## Dependency versions
KUBERNETES_VERSION ?= 1.30.0
BINDIR := $(shell pwd)/bin
SUDO := sudo
CURL := curl -sSLf

MINIKUBE_VERSION := v1.33.1
MINIKUBE_FEATURE_GATES="ReadWriteOncePod=true"
MINIKUBE_HOME = $(BINDIR)
export MINIKUBE_HOME

.PHONY: setup
setup:
	mkdir -p $(BINDIR)
	$(SUDO) apt-get update
	DEBIAN_FRONTEND=noninteractive $(SUDO) apt-get install -y --no-install-recommends conntrack unzip lvm2 xfsprogs thin-provisioning-tools patch
	$(SUDO) $(CURL) -o $(BINDIR)/minikube https://github.com/kubernetes/minikube/releases/download/$(MINIKUBE_VERSION)/minikube-linux-amd64
	$(SUDO) chmod a+x $(BINDIR)/minikube

.PHONY: launch
launch:
	$(SUDO) -E $(BINDIR)/minikube start \
    		--vm-driver=none \
    		--kubernetes-version=v$(KUBERNETES_VERSION) \
    		--extra-config=kubelet.read-only-port=10255 \
    		--feature-gates=$(MINIKUBE_FEATURE_GATES) \
    		--cni=calico
	$(SUDO) chown -R $$USER $$HOME/.kube $(MINIKUBE_HOME)/.minikube
	$(SUDO) chmod -R a+r $$HOME/.kube $(MINIKUBE_HOME)/.minikube
	$(SUDO) find $(MINIKUBE_HOME)/.minikube -name id_rsa -exec chmod 600 {} ';'

.PHONY: delete
delete:
	$(SUDO) -E $(BINDIR)/minikube delete || true
