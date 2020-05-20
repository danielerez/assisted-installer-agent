package assisted_installer_controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/eranco74/assisted-installer/src/k8s_client"

	"github.com/eranco74/assisted-installer/src/inventory_client"
	"github.com/eranco74/assisted-installer/src/ops"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "installer_test")
}

var _ = Describe("installer HostRoleMaster role", func() {
	var (
		l             = logrus.New()
		ctrl          *gomock.Controller
		mockops       *ops.MockOps
		mockbmclient  *inventory_client.MockInventoryClient
		mockk8sclient *k8s_client.MockK8SClient
		c             *controller
		hostIds       []string
	)
	hostIds = []string{"7916fa89-ea7a-443e-a862-b3e930309f65", "eb82821f-bf21-4614-9a3b-ecb07929f238", "b898d516-3e16-49d0-86a5-0ad5bd04e3ed"}
	l.SetOutput(ioutil.Discard)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockops = ops.NewMockOps(ctrl)
		mockbmclient = inventory_client.NewMockInventoryClient(ctrl)
		mockk8sclient = k8s_client.NewMockK8SClient(ctrl)
		hostIds = []string{"7916fa89-ea7a-443e-a862-b3e930309f65", "eb82821f-bf21-4614-9a3b-ecb07929f238", "b898d516-3e16-49d0-86a5-0ad5bd04e3ed"}
	})

	getInventoryNodes := func() []string {
		mockbmclient.EXPECT().GetHostsIds().Return(hostIds, nil).Times(1)
		return hostIds
	}

	udpateStatusSuccess := func(statuses []string, hostIds []string) {
		for i, status := range statuses {
			mockbmclient.EXPECT().UpdateHostStatus(status, hostIds[i]).Return(nil).Times(1)
		}
	}

	listNodes := func() {
		mockk8sclient.EXPECT().ListNodes().Return(GetKubeNodes(hostIds), nil).Times(1)
	}

	Context("Waiting for 3 nodes", func() {
		conf := ControllerConfig{
			ClusterID: "cluster-id",
			Host:      "https://bm-inventory.com",
			Port:      80,
		}
		BeforeEach(func() {
			c = NewController(l, conf, mockops, mockbmclient, mockk8sclient)
		})
		It("WaitAndUpdateNodesStatus happy flow", func() {
			udpateStatusSuccess([]string{done, done, done}, hostIds)
			getInventoryNodes()
			listNodes()
			c.WaitAndUpdateNodesStatus()

		})
		AfterEach(func() {
			ctrl.Finish()
		})
	})
	Context("Waiting for 3 nodes, will appear one by one", func() {
		conf := ControllerConfig{
			ClusterID: "cluster-id",
			Host:      "https://bm-inventory.com",
			Port:      80,
		}
		BeforeEach(func() {
			c = NewController(l, conf, mockops, mockbmclient, mockk8sclient)
			udpateStatusSuccess = func(statuses []string, hostIds []string) {
				for i, status := range statuses {
					mockbmclient.EXPECT().UpdateHostStatus(status, hostIds[i]).Return(nil).Times(1)
				}
			}
			hostIds = []string{"7916fa89-ea7a-443e-a862-b3e930309f65", "eb82821f-bf21-4614-9a3b-ecb07929f238", "b898d516-3e16-49d0-86a5-0ad5bd04e3ed"}
		})
		It("WaitAndUpdateNodesStatus one by one", func() {
			listNodes := func() {
				var hostIdsToReturn []string
				for _, host := range hostIds {
					hostIdsToReturn = append(hostIdsToReturn, host)
					mockk8sclient.EXPECT().ListNodes().Return(GetKubeNodes(hostIdsToReturn), nil).Times(1)
				}
			}

			udpateStatusSuccess([]string{done, done, done}, hostIds)
			getInventoryNodes()
			listNodes()
			c.WaitAndUpdateNodesStatus()

		})
		AfterEach(func() {
			ctrl.Finish()
		})
	})
	Context("UpdateStatusFails and then succeeds", func() {
		conf := ControllerConfig{
			ClusterID: "cluster-id",
			Host:      "https://bm-inventory.com",
			Port:      80,
		}
		BeforeEach(func() {
			c = NewController(l, conf, mockops, mockbmclient, mockk8sclient)
		})
		It("UpdateStatus fails and then succeeds", func() {
			udpateStatusSuccessFailureTest := func(statuses []string, hostIds []string) {
				for i, status := range statuses {
					mockbmclient.EXPECT().UpdateHostStatus(status, hostIds[i]).Return(fmt.Errorf("dummy")).Times(1)
					mockbmclient.EXPECT().UpdateHostStatus(status, hostIds[i]).Return(nil).Times(1)
				}
			}
			mockk8sclient.EXPECT().ListNodes().Return(GetKubeNodes(hostIds), nil).Times(4)
			udpateStatusSuccessFailureTest([]string{done, done, done}, hostIds)
			getInventoryNodes()
			c.WaitAndUpdateNodesStatus()

		})
		AfterEach(func() {
			ctrl.Finish()
		})
	})
	Context("ListNodes fails and then succeeds", func() {
		conf := ControllerConfig{
			ClusterID: "cluster-id",
			Host:      "https://bm-inventory.com",
			Port:      80,
		}
		BeforeEach(func() {
			c = NewController(l, conf, mockops, mockbmclient, mockk8sclient)
		})
		It("ListNodes fails and then succeeds", func() {
			listNodes := func() {
				mockk8sclient.EXPECT().ListNodes().Return(nil, fmt.Errorf("dummy")).Times(1)
				mockk8sclient.EXPECT().ListNodes().Return(GetKubeNodes(hostIds), nil).Times(1)
			}
			udpateStatusSuccess([]string{done, done, done}, hostIds)
			getInventoryNodes()
			listNodes()
			c.WaitAndUpdateNodesStatus()

		})
		AfterEach(func() {
			ctrl.Finish()
		})
	})
	Context("validating getInventoryNodes", func() {
		conf := ControllerConfig{
			ClusterID: "cluster-id",
			Host:      "https://bm-inventory.com",
			Port:      80,
		}
		BeforeEach(func() {
			c = NewController(l, conf, mockops, mockbmclient, mockk8sclient)
			getInventoryNodesTimeout = 1 * time.Second
		})
		It("inventory client fails and return result only on second run", func() {
			mockbmclient.EXPECT().GetHostsIds().Return(nil, fmt.Errorf("dummy")).Times(1)
			mockbmclient.EXPECT().GetHostsIds().Return(hostIds, nil).Times(1)
			nodesIds := c.getInventoryNodes()
			Expect(nodesIds).Should(Equal(hostIds))
		})
		AfterEach(func() {
			ctrl.Finish()
		})
	})
})

func GetKubeNodes(hostIds []string) *v1.NodeList {
	file, _ := ioutil.ReadFile("../../test_files/node.json")
	var node v1.Node
	_ = json.Unmarshal(file, &node)
	nodeList := &v1.NodeList{}
	for _, id := range hostIds {
		node.Status.NodeInfo.SystemUUID = id
		nodeList.Items = append(nodeList.Items, node)
	}
	return nodeList
}