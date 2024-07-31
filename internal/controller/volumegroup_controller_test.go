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
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jakobmoellerdev/lvm2go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/topolvm/topovgm/internal/lsblk"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	topolvmv1alpha1 "github.com/topolvm/topovgm/api/v1alpha1"
)

var _ = Describe("VolumeGroup Controller", func() {
	Context("happy path", func() {
		const resourceName = "test-resource"
		const resourceNamespace = "default"
		const nodeName = "test-node"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}

		ctx := context.Background()
		client := lvm2go.NewClient()

		loop := SetupLoopbackDevice()

		BeforeEach(func() {
			By("creating VolumeGroup resource")
			resource := &topolvmv1alpha1.VolumeGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: topolvmv1alpha1.VolumeGroupSpec{
					NodeName: nodeName,
					PhysicalVolumeSelector: topolvmv1alpha1.PhysicalVolumeSelector{{
						MatchLSBLK: []topolvmv1alpha1.LSBLKSelectorRequirement{{
							Key:      topolvmv1alpha1.LSBLKSelectorKey(lsblk.ColumnPath),
							Operator: topolvmv1alpha1.PVSelectorOpIn,
							Values:   []string{loop().Device()},
						}},
					}},
					Tags: []string{
						ValidLVMTag(GinkgoT().Name()),
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})
		AfterEach(func() {
			resource := &topolvmv1alpha1.VolumeGroup{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if errors.IsNotFound(err) {
				return
			}
			Expect(err).ToNot(HaveOccurred())
			GinkgoLogr.Info("summarizing the resource", "resource", resource)
			encoder := json.NewEncoder(GinkgoWriter)
			encoder.SetIndent("", "  ")
			Expect(encoder.Encode(resource)).To(Succeed())
		})

		var controllerReconciler *VolumeGroupReconciler
		BeforeEach(func() {
			By("initializing the controller reconciler")
			controllerReconciler = &VolumeGroupReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				LVM:      client,
				NodeName: nodeName,
			}
		})

		It("should successfully reconcile the CR", func() {
			By("reconciling the created CR", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			By("having set the volume group finalizer", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				Expect(resource.GetFinalizers()).To(ContainElement(VolumeGroupFinalizer))
			})

			By("reconciling the created CR again to sync it with the lvm state and triggering a condition", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			By("having a lvm2 volume group created", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				vg, err := client.VG(ctx, lvm2go.VolumeGroupName(resource.Status.Name))
				Expect(err).NotTo(HaveOccurred())
				Expect(vg).NotTo(BeNil())
			})

			By("having a VolumeGroupSyncedOnNode condition set to true", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				nodeCondition := meta.FindStatusCondition(
					resource.Status.Conditions,
					ConditionTypeVolumeGroupSyncedOnNode,
				)
				Expect(nodeCondition).NotTo(BeNil())
				Expect(nodeCondition.Status).To(Equal(metav1.ConditionTrue))
				Expect(nodeCondition.Reason).To(Equal(ReasonVolumeGroupSynced))
			})

			By("Delete the VolumeGroup CR and make it drop the finalizer", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				if errors.IsNotFound(err) {
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, typeNamespacedName, &topolvmv1alpha1.VolumeGroup{})).Should(Satisfy(errors.IsNotFound))
			})
		})
	})

	Context("failure path - external modifications", func() {
		const resourceName = "test-resource"
		const resourceNamespace = "default"
		const nodeName = "test-node"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}

		ctx := context.Background()
		client := lvm2go.NewClient()

		loopA := SetupLoopbackDevice()
		loopB := SetupLoopbackDevice()

		BeforeEach(func() {
			By("creating VolumeGroup resource")
			resource := &topolvmv1alpha1.VolumeGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: topolvmv1alpha1.VolumeGroupSpec{
					NodeName: nodeName,
					PhysicalVolumeSelector: topolvmv1alpha1.PhysicalVolumeSelector{{
						MatchLSBLK: []topolvmv1alpha1.LSBLKSelectorRequirement{{
							Key:      topolvmv1alpha1.LSBLKSelectorKey(lsblk.ColumnPath),
							Operator: topolvmv1alpha1.PVSelectorOpIn,
							Values: []string{
								loopA().Device(),
								loopB().Device(),
							},
						}},
					}},
					Tags: []string{
						ValidLVMTag(GinkgoT().Name()),
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})
		AfterEach(func() {
			resource := &topolvmv1alpha1.VolumeGroup{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if errors.IsNotFound(err) {
				return
			}
			Expect(err).ToNot(HaveOccurred())
			GinkgoLogr.Info("summarizing the resource", "resource", resource)
			encoder := json.NewEncoder(GinkgoWriter)
			encoder.SetIndent("", "  ")
			Expect(encoder.Encode(resource)).To(Succeed())
		})

		var controllerReconciler *VolumeGroupReconciler
		BeforeEach(func() {
			By("initializing the controller reconciler")
			controllerReconciler = &VolumeGroupReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				LVM:      client,
				NodeName: nodeName,
			}
		})

		It("should recover from a removal of a device", func() {
			By("reconciling the created CR", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			By("having set the volume group finalizer", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				Expect(resource.GetFinalizers()).To(ContainElement(VolumeGroupFinalizer))
			})

			By("reconciling the created CR again to sync it with the lvm state and triggering a condition", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			By("having a lvm2 volume group created", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				vg, err := client.VG(ctx, lvm2go.VolumeGroupName(resource.Status.Name))
				Expect(err).NotTo(HaveOccurred())
				Expect(vg).NotTo(BeNil())
			})

			By("having a VolumeGroupSyncedOnNode condition set to true", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				nodeCondition := meta.FindStatusCondition(
					resource.Status.Conditions,
					ConditionTypeVolumeGroupSyncedOnNode,
				)
				Expect(nodeCondition).NotTo(BeNil())
				Expect(nodeCondition.Status).To(Equal(metav1.ConditionTrue))
				Expect(nodeCondition.Reason).To(Equal(ReasonVolumeGroupSynced))
			})

			By("removing a device outside of the controller that can no longer be picked up", func() {
				Expect(loopB().Close()).To(Succeed())

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})

				Expect(err).To(Satisfy(lvm2go.IsLVMErrVGMissingPVs))

				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())

				Expect(resource.Status.PhysicalVolumeCount).To(BeEquivalentTo(2))
			})

			By("adjusting the DeviceLossSynchronizationPolicy to Remove", func() {
				resource := &topolvmv1alpha1.VolumeGroup{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
				resource.Spec.DeviceLossSynchronizationPolicy = topolvmv1alpha1.DeviceLossSynchronizationPolicyRemoveMissing
				Expect(k8sClient.Update(ctx, resource)).To(Succeed())

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func ValidLVMTag(name string) string {
	name = strings.ToLower(name)
	return strings.NewReplacer(" ", "-", "_", "-").Replace(name)
}

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

func SetupLoopbackDevice() func() lvm2go.LoopbackDevice {
	var loop lvm2go.LoopbackDevice
	BeforeEach(func() {
		By("preparing a new loopback device")
		backingFilePath := filepath.Join(GinkgoT().TempDir(), fmt.Sprintf("%s.img", NewNonDeterministicTestID(GinkgoT())))
		var err error
		loop, err = lvm2go.CreateLoopbackDevice(lvm2go.MustParseSize("10M"))
		Expect(err).NotTo(HaveOccurred())
		Expect(loop.FindFree()).To(Succeed())
		Expect(loop.SetBackingFile(backingFilePath)).To(Succeed())
		By(fmt.Sprintf("creating %q - backing file: %q", loop.Device(), loop.File()))
		Expect(loop.Open()).To(Succeed())
	})
	AfterEach(func() {
		By("cleaning up the loopback device")
		Expect(loop.Close()).To(Succeed())
	})
	return func() lvm2go.LoopbackDevice {
		return loop
	}
}
