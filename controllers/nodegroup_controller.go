/*
Copyright 2021 ttlv.

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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/google/martian/log"
	nodegroupv1alpha1 "github.com/ttlv/NodeGroup/api/v1alpha1"
	"github.com/ttlv/NodeGroup/model"
	nmv1alpha1 "github.com/ttlv/nodemaintenances/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

// NodeGroupReconciler reconciles a NodeGroup object
type NodeGroupReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=edge.harmonycloud.cn,resources=nodegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edge.harmonycloud.cn,resources=nodegroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edge.harmonycloud.cn,resources=nodemaintenance,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edge.harmonycloud.cn,resources=nodemaintenance/status,verbs=get;update;patch

func (r *NodeGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		nodeGroupList = &nodegroupv1alpha1.NodeGroupList{}
		//nodeGroupSelectors []string
		nodeList       = &v1.NodeList{}
		nodeGroupLists = []model.NodeGroupList{}
		ctx            = context.Background()
	)

	if err := r.List(ctx, nodeList); err != nil {
		log.Errorf("err is: %v and unable to fetch nodemaintenance", err)
	} else {
		for _, item := range nodeList.Items {
			var (
				nodeMaintenancesName, nodeGroupName string
				ready                               = "false"
			)
			for _, value := range item.Status.Conditions {
				if value.Type != "Ready" {
					continue
				}
				if value.Status == "True" {
					ready = "true"
				}
			}
			// 根据节点名字去关联nm对象,节点的命名格式为node-uniqueId,nodemaintenance的格式为nodemaintenances-uniqueId
			for key, value := range item.Labels {
				// kubeedge节点一定有nm对象
				if key == "kubernetes.io/hostname" {
					if strings.Contains(value, "node-") {
						nm := &nmv1alpha1.NodeMaintenance{}
						// 判断是否存在nm对象
						if err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: fmt.Sprintf("nodemaintenances-%v", strings.Split(value, "-")[1])}, nm); err == nil {
							nodeMaintenancesName = nm.Name
						}
					}
				}
				if strings.Contains(key, "name.nodegroup.edge.harmonycloud.cn/") {
					nodeGroupName = strings.Split(key, "/")[1]
				}
			}
			if nodeMaintenancesName != "" {
				nodeMaintenance := &nmv1alpha1.NodeMaintenance{}
				if err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: nodeMaintenancesName}, nodeMaintenance); err != nil {
					log.Errorf("err is: %v and unable to fetch nm", err)
					continue
				}
				nodeGroupLists = append(nodeGroupLists, model.NodeGroupList{
					NodeGroupName:       nodeGroupName,
					NodeName:            item.Name,
					NodeMaintenanceName: nodeMaintenance.Name,
					Ready:               ready,
					Maintenanable:       nodeMaintenance.Status.Conditions[0].Name,
				})
			} else {
				nodeGroupLists = append(nodeGroupLists, model.NodeGroupList{
					NodeGroupName: nodeGroupName,
					NodeName:      item.Name,
					Ready:         ready,
				})
			}
		}
	}
	if err := r.List(ctx, nodeGroupList); err != nil {
		log.Errorf("err is: %v and unable to fetch nodegroup", err)
	} else {
		for _, ngl := range nodeGroupList.Items {
			for _, temp := range nodeGroupLists {
				if ngl.Name == temp.NodeGroupName {
					var found bool
					for i := 0; i <= len(ngl.Status.NodeList)-1; i++ {
						if ngl.Status.NodeList[i].NodeName == temp.NodeName {
							found = true
							ngl.Status.NodeList[i].NodeName = temp.NodeName
							ngl.Status.NodeList[i].Maintenanable = temp.Maintenanable
							ngl.Status.NodeList[i].Ready = temp.Ready
							ngl.Status.NodeList[i].NodeMaintenanceName = temp.NodeMaintenanceName
						}
					}
					if !found {
						ngl.Status.NodeList = append(ngl.Status.NodeList, nodegroupv1alpha1.NodeList{
							NodeName:            temp.NodeName,
							NodeMaintenanceName: temp.NodeMaintenanceName,
							Ready:               temp.Ready,
							Maintenanable:       temp.Maintenanable,
						})
					}
				}
			}
			if err := r.Status().Update(ctx, &ngl); err != nil {
				fmt.Println("update error: ", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *NodeGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodegroupv1alpha1.NodeGroup{}).Watches(&source.Kind{Type: &nmv1alpha1.NodeMaintenance{}}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{Type: &v1.Node{}}, &handler.EnqueueRequestForObject{}).Complete(r)
}
