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
	_ "github.com/google/martian/log"
	nodegroupv1alpha1 "github.com/ttlv/NodeGroup/api/v1alpha1"
	"github.com/ttlv/NodeGroup/model"
	nmv1alpha1 "github.com/ttlv/nodemaintenances/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/apimachinery/pkg/types"
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
		//nodemaintenance     = &nmv1alpha1.NodeMaintenance{}
		ctx = context.Background()
		//_   = r.Log.WithValues("nodegroup", req.NamespacedName)
		//_   = r.Log.WithValues("nodemaintenances", req.NamespacedName)
	)

	if err := r.List(ctx, nodeList); err != nil {
		log.Errorf("err is: %v and unable to fetch nodemaintenance", err)
	} else {
		for _, item := range nodeList.Items {
			var (
				ready, nodeMaintenancesName, nodeGroupName string
				isEdge                                     bool
			)
			for _, value := range item.Status.Conditions {
				if value.Type == "Ready" {
					ready = "true"
				} else {
					ready = "false"
				}
			}
			for key, value := range item.Labels {
				// kubeedge节点一定有nm对象
				if key == "kubernetes.io/hostname" {
					if strings.Contains(value, "-") {
						nodeMaintenancesName = fmt.Sprintf("nodemaintenances-%v", strings.Split(value, "-")[1])
					}
				}
				if key == "node-role.kubernetes.io/edge" {
					isEdge = true
				}
				if strings.Contains(key, "name.nodegroup.edge.harmonycloud.cn/") {
					nodeGroupName = strings.Split(key, "/")[1]
				}
			}
			if isEdge {
				nodeMaintenance := &nmv1alpha1.NodeMaintenance{}
				if err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: nodeMaintenancesName}, nodeMaintenance); err != nil {
					log.Errorf("err is: %v and unable to fetch nm", err)
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
				var found bool
				for _, statusNodeList := range ngl.Status.NodeList {
					if statusNodeList.NodeName == temp.NodeName {
						found = true
						statusNodeList.NodeName = temp.NodeName
						statusNodeList.Maintenanable = temp.Maintenanable
						statusNodeList.Ready = temp.Ready
						statusNodeList.NodeMaintenanceName = temp.NodeMaintenanceName
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
