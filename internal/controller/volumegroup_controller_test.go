/*
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
*/

package controller

import (
	"context"
	"crypto/rand"
	"fmt"
	"hash"
	"hash/fnv"
	"path/filepath"
	"strconv"

	"github.com/jakobmoellerdev/lvm2go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	topolvmv1alpha1 "github.com/topolvm/topovgm/api/v1alpha1"
)

var _ = Describe("VolumeGroup Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const resourceNamespace = "default"
		const nodeName = "test-node"

		ctx := context.Background()

		client := lvm2go.NewClient()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		volumegroup := &topolvmv1alpha1.VolumeGroup{}

		var loop lvm2go.LoopbackDevice
		BeforeEach(func() {
			backingFilePath := filepath.Join(GinkgoT().TempDir(), fmt.Sprintf("%s.img", NewNonDeterministicTestID(GinkgoT())))
			By("creating a loopback device")
			var err error
			loop, err = lvm2go.CreateLoopbackDevice(lvm2go.MustParseSize("10M"))
			Expect(err).NotTo(HaveOccurred())
			Expect(loop.FindFree()).To(Succeed())
			Expect(loop.SetBackingFile(backingFilePath)).To(Succeed())
			Expect(loop.Open()).To(Succeed())
		})
		AfterEach(func() {
			By("cleaning up the loopback device")
			Expect(loop.Close()).To(Succeed())
		})

		BeforeEach(func() {
			By("creating the custom resource for the Kind VolumeGroup")
			err := k8sClient.Get(ctx, typeNamespacedName, volumegroup)
			if err != nil && errors.IsNotFound(err) {
				resource := &topolvmv1alpha1.VolumeGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: topolvmv1alpha1.VolumeGroupSpec{
						NodeName: nodeName,
						PVs: []string{
							loop.Device(),
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &topolvmv1alpha1.VolumeGroup{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the VolumeGroup CR")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the CR", func() {
			By("reconciling the created CR")
			controllerReconciler := &VolumeGroupReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				LVM:      client,
				NodeName: nodeName,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("having a VolumeGroupSyncedOnNode condition set to true")
			resource := &topolvmv1alpha1.VolumeGroup{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			nodeCondition := meta.FindStatusCondition(
				resource.Status.Conditions,
				ConditionTypeVolumeGroupSyncedOnNode,
			)
			Expect(nodeCondition).NotTo(BeNil())
			Expect(nodeCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(nodeCondition.Reason).To(Equal(ReasonVolumeGroupCreated))

			By("having a lvm2 volume group created")
			vg, err := client.VG(ctx, lvm2go.VolumeGroupName(resourceName))
			Expect(err).NotTo(HaveOccurred())
			Expect(vg).NotTo(BeNil())
		})
	})
})

type TestIDT interface {
	Fatal(args ...any)
}

func NewNonDeterministicTestID(t TestIDT) string {
	return strconv.Itoa(int(NewNonDeterministicTestHash(t).Sum32()))
}

func NewNonDeterministicTestHash(t TestIDT) hash.Hash32 {
	hashedTestName := fnv.New32()
	randomData := make([]byte, 32)
	if _, err := rand.Read(randomData); err != nil {
		t.Fatal(err)
	}
	if _, err := hashedTestName.Write(randomData); err != nil {
		t.Fatal(err)
	}
	return hashedTestName
}
