// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

const (
	allowlistCMname = "observability-metrics-custom-allowlist"
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

	It("[P2][Sev2][Observability][Stable] Should have metrics which defined in custom metrics allowlist (metricslist/g0)", func() {
		By("Adding custom metrics allowlist configmap")
		yamlB, err := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/metrics/allowlist"})
		Expect(err).ToNot(HaveOccurred())
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		By("Waiting for new added metrics on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes offset 1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Integration] Should have not metrics which have been marked for deletion (metricslist/g0)", func() {
		By("Waiting for deleted metrics disappear on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "rest_client_requests_total offset 1m", []string{`"__name__":"rest_client_requests_total"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})

	It("[P2][Sev2][Observability][Stable] Should have no metrics after custom metrics allowlist deleted (metricslist/g0)", func() {
		By("Deleting custom metrics allowlist configmap")
		Eventually(func() error {
			err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(allowlistCMname, &metav1.DeleteOptions{})
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(Succeed())

		By("Waiting for new added metrics disappear on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes offset 1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})

	AfterEach(func() {
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
		if testFailed {
			utils.PrintMCORelatedInfoForDebug(testOptions)
		} else {
			Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
		}
	})
})
