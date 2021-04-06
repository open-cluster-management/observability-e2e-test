// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

var (
	ThanosRuleName       = MCO_CR_NAME + "-thanos-rule"
	ThanosRuleRestarting = false
)

var _ = Describe("Observability:", func() {
	BeforeEach(func() {
		hubClient = utils.NewKubeClient(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)

		dynClient = utils.NewKubeClientDynamic(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)
	})
	statefulset := [...]string{MCO_CR_NAME + "-alertmanager", ThanosRuleName}
	configmap := [...]string{"thanos-ruler-default-rules", "thanos-ruler-custom-rules"}
	secret := "alertmanager-config"

	It("[P1][Sev1][Observability] Should have the expected statefulsets (alert/g0)", func() {
		By("Checking if STS: Alertmanager and observability-thanos-rule exist")
		for _, name := range statefulset {
			sts, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(sts.Spec.Template.Spec.Volumes)).Should(BeNumerically(">", 0))

			if sts.GetName() == MCO_CR_NAME+"-alertmanager" {
				By("The statefulset: " + sts.GetName() + " should have the appropriate secret mounted")
				Expect(sts.Spec.Template.Spec.Volumes[0].Secret.SecretName).To(Equal("alertmanager-config"))
			}

			if sts.GetName() == ThanosRuleName {
				By("The statefulset: " + sts.GetName() + " should have the appropriate configmap mounted")
				Expect(sts.Spec.Template.Spec.Volumes[0].ConfigMap.Name).To(Equal("thanos-ruler-default-rules"))
			}
		}
	})

	It("[P2][Sev2][Observability] Should have the expected configmap (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-default-rules is existed")
		cm, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(configmap[0], metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(cm.ResourceVersion).ShouldNot(BeEmpty())
		klog.V(3).Infof("Configmap %s does exist", configmap[0])
	})

	It("[P3][Sev3][Observability] Should not have the CM: thanos-ruler-custom-rules (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-custom-rules not existed")
		_, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(configmap[1], metav1.GetOptions{})

		if err == nil {
			err = fmt.Errorf("%s exist within the namespace env", configmap[1])
			Expect(err).NotTo(HaveOccurred())
		}

		Expect(err).To(HaveOccurred())
		klog.V(3).Infof("Configmap %s does not exist", configmap[1])
	})

	It("[P1][Sev1][Observability] Should have the expected secret (alert/g0)", func() {
		By("Checking if SECRETS: alertmanager-config is existed")
		secret, err := hubClient.CoreV1().Secrets(MCO_NAMESPACE).Get(secret, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.GetName()).To(Equal("alertmanager-config"))
		klog.V(3).Infof("Successfully got secret: %s", secret.GetName())
	})

	It("[P2][Sev2][Observability] Should have custom alert generated (alert/g0)", func() {
		By("Creating custom alert rules")
		_, oldSts := utils.GetStatefulSet(testOptions, true, ThanosRuleName, MCO_NAMESPACE)

		yamlB, err := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/alerts/custom_rules_valid"})
		Expect(err).NotTo(HaveOccurred())
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		By("Wait for thanos rule pods are restarted and ready")
		// ensure the thanos rule pods are restarted successfully before processing
		Eventually(func() error {
			if !ThanosRuleRestarting {
				_, newSts := utils.GetStatefulSet(testOptions, true, ThanosRuleName, MCO_NAMESPACE)

				if oldSts.GetResourceVersion() == newSts.GetResourceVersion() {
					return fmt.Errorf("The %s is not being restarted in 3 minutes", ThanosRuleName)
				} else {
					ThanosRuleRestarting = true
				}
			}

			err = utils.CheckThanosRulePodReady(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())

		var labelName, labelValue string
		labels, err := kustomize.GetLabels(yamlB)
		Expect(err).NotTo(HaveOccurred())
		for labelName = range labels.(map[string]interface{}) {
			labelValue = labels.(map[string]interface{})[labelName].(string)
		}

		By("Checking alert generated")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, `ALERTS{`+labelName+`="`+labelValue+`"} offset 1m`,
				[]string{`"__name__":"ALERTS"`, `"` + labelName + `":"` + labelValue + `"`})
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability] Should modify the SECRET: alertmanager-config (alert/g0)", func() {
		By("Editing the secret, we should be able to add the third partying tools integrations")
		secret := utils.CreateCustomAlertConfigYaml(testOptions.HubCluster.BaseDomain)

		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, secret)).NotTo(HaveOccurred())
		klog.V(3).Infof("Successfully modified the secret: alertmanager-config")
	})

	It("[P2][Sev2][Observability] Should have custom alert updated (alert/g0)", func() {
		By("Updating custom alert rules")

		yamlB, _ := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/alerts/custom_rules_invalid"})
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		var labelName, labelValue string
		labels, _ := kustomize.GetLabels(yamlB)
		for labelName = range labels.(map[string]interface{}) {
			labelValue = labels.(map[string]interface{})[labelName].(string)
		}

		By("Checking alert generated")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, `ALERTS{`+labelName+`="`+labelValue+`"} offset 1m`,
				[]string{`"__name__":"ALERTS"`, `"` + labelName + `":"` + labelValue + `"`})
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})

	It("[P2][Sev2][Observability] delete the customized rules (alert/g0)", func() {
		_, oldSts := utils.GetStatefulSet(testOptions, true, ThanosRuleName, MCO_NAMESPACE)

		Eventually(func() error {
			err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(configmap[1], &metav1.DeleteOptions{})
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(Succeed())

		ThanosRuleRestarting = false
		By("Wait for thanos rule pods are restarted and ready")
		// ensure the thanos rule pods are restarted successfully before processing
		Eventually(func() error {
			if !ThanosRuleRestarting {
				_, newSts := utils.GetStatefulSet(testOptions, true, ThanosRuleName, MCO_NAMESPACE)

				if oldSts.GetResourceVersion() == newSts.GetResourceVersion() {
					return fmt.Errorf("The %s is not being restarted in 3 minutes", ThanosRuleName)
				} else {
					ThanosRuleRestarting = true
				}
			}

			err = utils.CheckThanosRulePodReady(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())

		klog.V(3).Infof("Successfully deleted CM: thanos-ruler-custom-rules")
	})

	AfterEach(func() {
		if testFailed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		} else {
			Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
