package tests

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
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

	It("should have not the expected MCO addon pods (addon/g0)", func() {
		By("Modifying MCO cr to disable observabilityaddon")
		err := utils.ModifyMCOAddonSpecMetrics(testOptions, false)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for MCO addon components scales to 0")
		Eventually(func() error {
			addonLabel := "component=metrics-collector"
			var podList, _ = hubClient.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{LabelSelector: addonLabel})
			if len(podList.Items) != 0 {
				return fmt.Errorf("Failed to disable observability addon")
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
		clusters, err := dynClient.Resource(utils.NewOCMManagedClustersGVR()).List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		for _, cluster := range clusters.Items {
			clusterName := cluster.Object["metadata"].(map[string]interface{})["name"].(string)
			Eventually(func() string {
				mco, err := dynClient.Resource(utils.NewMCOAddonGVR()).Namespace(string(clusterName)).Get("observability-addon", metav1.GetOptions{})
				if err != nil {
					panic(err.Error())
				}
				if mco.Object["status"] != nil {
					return mco.Object["status"].(map[string]interface{})["conditions"].([]interface{})[0].(map[string]interface{})["message"].(string)
				} else {
					return ""
				}
<<<<<<< HEAD
			}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Equal("enableMetrics is set to False"))
=======
				return mco.Object["status"].(map[string]interface{})["conditions"].([]interface{})[0].(map[string]interface{})["message"].(string)
			}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Equal("enableMetrics is set to False"))
>>>>>>> revert test timeout
		}
	})

	It("should have not metric data (addon/g0)", func() {
		By("Waiting for check no metric data in grafana console")
		Eventually(func() error {
			err, hasMetric := utils.ContainManagedClusterMetric(testOptions, "node_memory_MemAvailable_bytes", "90s", []string{`"__name__":"node_memory_MemAvailable_bytes"`})
			if err != nil && !hasMetric && strings.Contains(err.Error(), "Failed to find metric name from response") {
				return nil
			}
			return fmt.Errorf("Check no metric data in grafana console error: %v", err)
<<<<<<< HEAD
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(Succeed())
=======
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
>>>>>>> revert test timeout

		By("Modifying MCO cr to enalbe observabilityaddon")
		err := utils.ModifyMCOAddonSpecMetrics(testOptions, true)
		Expect(err).ToNot(HaveOccurred())
		By("Waiting for MCO addon components ready")
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if len(podList.Items) == 1 && err == nil {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("should not set interval to values beyond scope (addon/g0)", func() {
		By("Set interval to 14")
		Eventually(func() bool {
			err := utils.ModifyMCOAddonSpecInterval(testOptions, int64(14))
			if strings.Contains(err.Error(), "Invalid value: 15") {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Set interval to 3601")
		Eventually(func() bool {
			err := utils.ModifyMCOAddonSpecInterval(testOptions, int64(3601))
			if strings.Contains(err.Error(), "Invalid value: 3600") {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("should have not the expected MCO addon pods when disable observability from managedcluster (addon/g0)", func() {
		if !utils.IsCanaryEnvironment(testOptions) {
			Skip("Modifying managedcluster cr to disable observability")
		}
		By("Modifying managedcluster cr to disable observability")
		Eventually(func() error {
			return utils.UpdateObservabilityFromManagedCluster(testOptions, false)
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("Waiting for MCO addon components scales to 0")
		Eventually(func() bool {
			_, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if len(podList.Items) == 0 && err == nil {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Modifying managedcluster cr to enable observability")
		Eventually(func() error {
			return utils.UpdateObservabilityFromManagedCluster(testOptions, true)
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("Waiting for MCO addon components ready")
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if len(podList.Items) == 1 && err == nil {
				return true
			}
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
