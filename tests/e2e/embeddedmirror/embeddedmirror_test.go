package embeddedmirror

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/k3s-io/k3s/tests"
	"github.com/k3s-io/k3s/tests/e2e"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Valid nodeOS:
// bento/ubuntu-24.04, opensuse/Leap-15.6.x86_64
// eurolinux-vagrant/rocky-8, eurolinux-vagrant/rocky-9,
var nodeOS = flag.String("nodeOS", "bento/ubuntu-24.04", "VM operating system")
var serverCount = flag.Int("serverCount", 1, "number of server nodes")
var agentCount = flag.Int("agentCount", 1, "number of agent nodes")
var ci = flag.Bool("ci", false, "running on CI")
var local = flag.Bool("local", false, "deploy a locally built K3s binary")

// Environment Variables Info:
// E2E_RELEASE_VERSION=v1.23.1+k3s2 (default: latest commit from master)
// E2E_REGISTRY: true/false (default: false)

func Test_E2EEmbeddedMirror(t *testing.T) {
	RegisterFailHandler(Fail)
	flag.Parse()
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Embedded Mirror Test Suite", suiteConfig, reporterConfig)
}

var tc *e2e.TestConfig

var _ = ReportAfterEach(e2e.GenReport)

var _ = Describe("Verify Create", Ordered, func() {
	Context("Cluster :", func() {
		It("Starts up with no issues", func() {
			var err error
			if *local {
				tc, err = e2e.CreateLocalCluster(*nodeOS, *serverCount, *agentCount)
			} else {
				tc, err = e2e.CreateCluster(*nodeOS, *serverCount, *agentCount)
			}
			Expect(err).NotTo(HaveOccurred(), e2e.GetVagrantLog(err))
			By("CLUSTER CONFIG")
			By("OS: " + *nodeOS)
			By(tc.Status())

		})
		It("Checks node and pod status", func() {
			By("Fetching Nodes status")
			Eventually(func() error {
				return tests.NodesReady(tc.KubeconfigFile, e2e.VagrantSlice(tc.AllNodes()))
			}, "620s", "5s").Should(Succeed())

			By("Fetching pod status")
			Eventually(func() error {
				e2e.DumpPods(tc.KubeconfigFile)
				return tests.AllPodsUp(tc.KubeconfigFile)
			}, "620s", "10s").Should(Succeed())
		})
		It("Should create and validate deployment with embedded registry mirror using image tag", func() {
			res, err := e2e.RunCommand("kubectl create deployment my-webpage-1 --image=docker.io/library/nginx:1.25.3")
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())

			patchCmd := fmt.Sprintf(`kubectl patch deployment my-webpage-1 --patch '{"spec":{"replicas":%d,"revisionHistoryLimit":0,"strategy":{"type":"Recreate", "rollingUpdate": null},"template":{"spec":{"affinity":{"podAntiAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchExpressions":[{"key":"app","operator":"In","values":["my-webpage-1"]}]},"topologyKey":"kubernetes.io/hostname"}]}}}}}}'`, *serverCount+*agentCount)
			res, err = e2e.RunCommand(patchCmd)
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())

			res, err = e2e.RunCommand("kubectl rollout status deployment my-webpage-1 --watch=true --timeout=360s")
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create and validate deployment with embedded registry mirror using image digest for existing tag", func() {
			res, err := e2e.RunCommand("kubectl create deployment my-webpage-2 --image=docker.io/library/nginx:nginx@sha256:c7a6ad68be85142c7fe1089e48faa1e7c7166a194caa9180ddea66345876b9d2")
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())

			patchCmd := fmt.Sprintf(`kubectl patch deployment my-webpage-2 --patch '{"spec":{"replicas":%d,"revisionHistoryLimit":0,"strategy":{"type":"Recreate", "rollingUpdate": null},"template":{"spec":{"affinity":{"podAntiAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchExpressions":[{"key":"app","operator":"In","values":["my-webpage-2"]}]},"topologyKey":"kubernetes.io/hostname"}]}}}}}}'`, *serverCount+*agentCount)
			res, err = e2e.RunCommand(patchCmd)
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())

			res, err = e2e.RunCommand("kubectl rollout status deployment my-webpage-2 --watch=true --timeout=360s")
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create and validate deployment with embedded registry mirror using image digest without corresponding tag", func() {
			res, err := e2e.RunCommand("kubectl create deployment my-webpage-3 --image=docker.io/library/nginx@sha256:b4af4f8b6470febf45dc10f564551af682a802eda1743055a7dfc8332dffa595")
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())

			patchCmd := fmt.Sprintf(`kubectl patch deployment my-webpage-3 --patch '{"spec":{"replicas":%d,"revisionHistoryLimit":0,"strategy":{"type":"Recreate", "rollingUpdate": null},"template":{"spec":{"affinity":{"podAntiAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchExpressions":[{"key":"app","operator":"In","values":["my-webpage-3"]}]},"topologyKey":"kubernetes.io/hostname"}]}}}}}}'`, *serverCount+*agentCount)
			res, err = e2e.RunCommand(patchCmd)
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())

			res, err = e2e.RunCommand("kubectl rollout status deployment my-webpage-3 --watch=true --timeout=360s")
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should expose embedded registry metrics", func() {
			grepCmd := fmt.Sprintf("kubectl get --raw /api/v1/nodes/%s/proxy/metrics | grep -F 'spegel_advertised_images{registry=\"docker.io\"}'", tc.Servers[0])
			res, err := e2e.RunCommand(grepCmd)
			fmt.Println(res)
			Expect(err).NotTo(HaveOccurred())
		})
		It("Should cleanup deployments", func() {
			_, err := e2e.RunCommand("kubectl delete deployment my-webpage-1 my-webpage-2 my-webpage-3")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var failed bool
var _ = AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = AfterSuite(func() {
	if failed {
		Expect(e2e.SaveJournalLogs(tc.AllNodes())).To(Succeed())
		Expect(e2e.TailPodLogs(50, tc.AllNodes())).To(Succeed())
	} else {
		Expect(e2e.GetCoverageReport(tc.AllNodes())).To(Succeed())
	}
	if !failed || *ci {
		Expect(e2e.DestroyCluster()).To(Succeed())
		Expect(os.Remove(tc.KubeconfigFile)).To(Succeed())
	}
})
