package operators

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	maoDeployment        = "machine-api-operator"
	maoManagedDeployment = "machine-api-controllers"
)

var _ = Describe("[Feature:Operators] Machine API operator deployment should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.IsDeploymentAvailable(client, maoDeployment, framework.MachineAPINamespace)).To(BeTrue())
	})

	It("reconcile controllers deployment", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		initialDeployment, err := framework.GetDeployment(client, maoManagedDeployment, framework.MachineAPINamespace)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

		By(fmt.Sprintf("deleting deployment %q", maoManagedDeployment))
		Expect(framework.DeleteDeployment(client, initialDeployment)).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

		By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
		Expect(framework.IsDeploymentSynced(client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())
	})

	It("maintains deployment spec", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		initialDeployment, err := framework.GetDeployment(client, maoManagedDeployment, framework.MachineAPINamespace)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

		changedDeployment := initialDeployment.DeepCopy()
		changedDeployment.Spec.Replicas = pointer.Int32Ptr(0)

		By(fmt.Sprintf("updating deployment %q", maoManagedDeployment))
		Expect(framework.UpdateDeployment(client, maoManagedDeployment, framework.MachineAPINamespace, changedDeployment)).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
		Expect(framework.IsDeploymentSynced(client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

		By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

	})

	It("reconcile mutating webhook configuration", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("reconcile validating webhook configuration", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("recover after validating webhook configuration deletion", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		// Record the UID of the current ValidatingWebhookConfiguration
		initial, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(initial).ToNot(BeNil())
		initialUID := initial.GetUID()

		Expect(framework.DeleteValidatingWebhookConfiguration(client, initial)).To(Succeed())

		// Ensure that either UID changes (to show a new object) or that the existing object is gone
		key := runtimeclient.ObjectKey{Name: initial.Name}
		Eventually(func() (apitypes.UID, error) {
			current := &admissionregistrationv1.ValidatingWebhookConfiguration{}
			if err := client.Get(context.Background(), key, current); err != nil && !apierrors.IsNotFound(err) {
				return "", err
			}
			return current.GetUID(), nil
		}).ShouldNot(Equal(initialUID))

		// Ensure that a new object has been created and matches expectations
		Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("recover after mutating webhook configuration deletion", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		// Record the UID of the current MutatingWebhookConfiguration
		initial, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(initial).ToNot(BeNil())
		initialUID := initial.GetUID()

		Expect(framework.DeleteMutatingWebhookConfiguration(client, initial)).To(Succeed())

		// Ensure that either UID changes (to show a new object) or that the existing object is gone
		key := runtimeclient.ObjectKey{Name: initial.Name}
		Eventually(func() (apitypes.UID, error) {
			current := &admissionregistrationv1.MutatingWebhookConfiguration{}
			if err := client.Get(context.Background(), key, current); err != nil && !apierrors.IsNotFound(err) {
				return "", err
			}
			return current.GetUID(), nil
		}).ShouldNot(Equal(initialUID))

		// Ensure that a new object has been created and matches expectations
		Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("maintains spec after mutating webhook configuration change and preserve caBundle", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		initial, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(initial).ToNot(BeNil())

		toUpdate := initial.DeepCopy()
		for _, webhook := range toUpdate.Webhooks {
			webhook.ClientConfig.CABundle = []byte("test")
			webhook.AdmissionReviewVersions = []string{"test"}
		}

		Expect(framework.UpdateMutatingWebhookConfiguration(client, toUpdate)).To(Succeed())

		Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())

		updated, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).ToNot(BeNil())

		for i, webhook := range updated.Webhooks {
			Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
		}
	})

	It("maintains spec after validating webhook configuration change and preserve caBundle", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		initial, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(initial).ToNot(BeNil())

		toUpdate := initial.DeepCopy()
		for _, webhook := range toUpdate.Webhooks {
			webhook.ClientConfig.CABundle = []byte("test")
			webhook.AdmissionReviewVersions = []string{"test"}
		}

		Expect(framework.UpdateValidatingWebhookConfiguration(client, toUpdate)).To(Succeed())

		Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())

		updated, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).ToNot(BeNil())

		for i, webhook := range updated.Webhooks {
			Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
		}
	})

})

var _ = Describe("[Feature:Operators] Machine API cluster operator status should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.IsStatusAvailable(client, "machine-api")).To(BeTrue())
	})
})
