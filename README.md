# TopoVGM

TopoVGM (Topology Volume Group Manager) is a Kubernetes Operator that manages the creation and deletion of Logical Volume Manager (LVM) Volume Group on Kubernetes nodes as a daemon. It is intended to be used in conjunction with the [TopoLVM](https://github.com/topolvm/topolvm) project, but can be used independently as well. Its heart lies within its ability to establish a syncing procedure for volume groups across multiple nodes in a Kubernetes cluster.

My final goal is to contribute this project to the TopoLVM project community as I believe that is where it can mature into a more robust and feature-rich project.

## Description

TopoVGM is a Kubernetes Operator that manages the creation and deletion of Logical Volume Manager (LVM) Volume Group on Kubernetes nodes as a daemon.

To do this it makes of a Custom Resource Definition (CRD) called `VolumeGroup` which is a representation of the Volume Group that needs to be created on the node. The operator watches for the creation of this CRD and then creates the Volume Group on the node.

The operator also watches for the deletion of the CRD and then deletes the Volume Group on the node.

Here is an example of the `VolumeGroup` CRD that will attempt to use all loop devices on the node as well as a custom storage device:

```yaml
apiVersion: topolvm.io/v1alpha1
kind: VolumeGroup
metadata:
  labels:
  name: vg
spec:
  nodeName: my-node
  physicalVolumeSelector:
    - matchLSBLK:
        - key: TYPE
          operator: In
          values:
            - loop
    - matchLSBLK:
        - key: PATH
          operator: In
          values:
            - /dev/sda
```

While this is a simple example, the `VolumeGroup` CRD can be customized to match any specific requirements that you may have.
Almost all of the vgcreate / vgchange commands you are used to from the command line can be represented in the `VolumeGroup` CRD.

You will notice that the VolumeGroup exposes a significant Status that can be used both for extension of functionality, debugging or monitoring. Here is an example for that

```yaml
apiVersion: topolvm.io/v1alpha1
kind: VolumeGroup
metadata:
  finalizers:
  - topolvm.io/volumegroup-removal-on-node
  generation: 1
  name: vg1
  namespace: default
  resourceVersion: "175375"
  uid: 2903b9fa-5f09-41ec-b5f6-dcb60fe7e261
spec:
  allocationPolicy: normal
  deviceLossSynchronizationPolicy: Fail
  deviceRemovalVolumePolicy: MoveAndReduce
  nodeName: crc
  physicalExtentSize: 4Mi
  physicalVolumeSelector:
  - matchLSBLK:
    - key: TYPE
      operator: In
      values:
      - loop
  tags:
  - topovgm
  zero: true
status:
  attributes: wz--n-
  conditions:
  - lastTransitionTime: "2024-07-31T20:05:51Z"
    message: The volume group is present on the node and discoverable in the lvm2 subsystem.
    observedGeneration: 1
    reason: VolumeGroupSynced
    status: "True"
    type: VolumeGroupSyncedOnNode
  extentCount: 510
  extentSize: "4194304"
  free: "2139095040"
  metadataAreaCount: 2
  metadataAreaUsedCount: 2
  name: 2903b9fa-5f09-41ec-b5f6-dcb60fe7e261
  physicalVolumeCount: 2
  physicalVolumes:
  - attributes: a--
    deviceID: /lblock0
    deviceSize: "1073741824"
    free: "1069547520"
    major: 7
    metadataAreaCount: 1
    metadataAreaFree: "520192"
    metadataAreaSize: "1044480"
    metadataAreaUsedCount: 1
    minor: 0
    name: /dev/loop0
    physicalExtentStart: "1048576"
    size: "1069547520"
    used: "0"
    uuid: 5mzDdg-Yn5e-lLbQ-9Emj-Syzg-seVC-BJdHz0
  - attributes: a--
    deviceID: /lblock1
    deviceSize: "1073741824"
    free: "1069547520"
    major: 7
    metadataAreaCount: 1
    metadataAreaFree: "520192"
    metadataAreaSize: "1044480"
    metadataAreaUsedCount: 1
    minor: 1
    name: /dev/loop1
    physicalExtentStart: "1048576"
    size: "1069547520"
    used: "0"
    uuid: X1ksoF-9xV4-ECUU-Ygvz-YfK2-fYeb-xMLv6A
  seqno: 1
  size: "2139095040"
  tags:
  - topovgm
  uuid: izMq2s-Sgbo-JOfd-kY4a-lDA1-j1kQ-iF7rFL
```


## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.30+.
- Access to a Kubernetes v1.30+ cluster.
- lsblk from util-linux 2.39.4+
- lvm2 version 2.03.11+ (ideally 2.03.23) on the node

To install the node dependencies, you can run the following command:

#### For Debian-based systems
```sh
sudo apt-get install -y lvm2 util-linux
```

#### For CentOS / Fedora / RHEL
```sh
sudo dnf install -y lvm2 util-linux
```

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/topovgm:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/topovgm:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/topovgm:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/topovgm/<tag or branch>/dist/install.yaml
```

## Contributing

This project is in early stages of development and is open to contributions of any kind. I am looking for help in the following areas:
- Testing
- Documentation
- Code Contributions
- Bug Reports
- Feature Requests
- Feedback
- Code Reviews

Please feel free to open an issue or a pull request if you would like to contribute.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

