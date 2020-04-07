/*


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
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ethereumv1alpha1 "github.com/mfarghaly/kotal/api/v1alpha1"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ethereum.kotal.io,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ethereum.kotal.io,resources=networks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;create;update

// Reconcile reconciles ethereum networks
func (r *NetworkReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("network", req.NamespacedName)

	var network ethereumv1alpha1.Network

	// Get desired ethereum network
	if err := r.Client.Get(ctx, req.NamespacedName, &network); err != nil {
		log.Error(err, "Unable to fetch Ethereum Network")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("deleting all redundant nodes")
	if err := r.deleteRedundantNodes(ctx, network.Spec.Nodes, req.Namespace); err != nil {
		return ctrl.Result{}, err
	}

	for _, node := range network.Spec.Nodes {
		if err := r.reconcileNode(ctx, &node, &network); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deleteRedundantNode deletes all nodes that has been removed from spec
func (r *NetworkReconciler) deleteRedundantNodes(ctx context.Context, nodes []ethereumv1alpha1.Node, ns string) error {
	var deps appsv1.DeploymentList
	names := map[string]bool{}

	// all node names in the spec
	for _, node := range nodes {
		names[node.Name] = true
	}

	// all nodes deployments that's currently running
	if err := r.Client.List(ctx, &deps, client.MatchingLabels{"app": "node"}); err != nil {
		r.Log.Error(err, "unable to list all node deployments")
		return err
	}

	for _, dep := range deps.Items {
		name := dep.GetName()
		if exist := names[name]; !exist {
			r.Log.Info(fmt.Sprintf("node (%s) deployment doesn't exist anymore in the spec", name))
			r.Log.Info(fmt.Sprintf("deleting node (%s) deployment", name))

			if err := r.Client.Delete(ctx, &dep); err != nil {
				r.Log.Error(err, fmt.Sprintf("unable to delete node (%s) deployment", name))
				return err
			}
		}
	}

	return nil
}

// reconcileNode create a new node deployment if it doesn't exist
// updates existing deployments if node spec changed
func (r *NetworkReconciler) reconcileNode(ctx context.Context, node *ethereumv1alpha1.Node, network *ethereumv1alpha1.Network) error {
	log := r.Log.WithValues("node", node.Name)
	dep := &appsv1.Deployment{}
	ns := network.ObjectMeta.Namespace

	err := r.Client.Get(ctx, client.ObjectKey{
		Name:      node.Name,
		Namespace: ns,
	}, dep)

	notFound := errors.IsNotFound(err)

	if err != nil && !notFound {

		log.Error(err, fmt.Sprintf("unable to find node (%s) deployment", node.Name))
		return err

	}

	if err := r.createOrUpdateNode(ctx, node, network, ns, !notFound); err != nil {
		return err
	}

	return nil
}

// createOrUpdateNode creates a node deployment if it doesn't exist
// updates existing deployment if node spec changed
func (r *NetworkReconciler) createOrUpdateNode(ctx context.Context, node *ethereumv1alpha1.Node, network *ethereumv1alpha1.Network, ns string, found bool) error {
	log := r.Log.WithValues("node", node.Name)
	args := r.createArgsForClient(node, network.Spec.Join)
	newDep := r.createDeploymentForNode(node, ns, args)

	if err := ctrl.SetControllerReference(network, &newDep, r.Scheme); err != nil {
		log.Error(err, "Unable to set controller reference")
		return err
	}

	if found {
		log.Info(fmt.Sprintf("updating node (%s) deployment", node.Name))

		if err := r.Client.Update(ctx, &newDep); err != nil {
			log.Error(err, fmt.Sprintf("unable to update node (%s) deployment", node.Name))
			return err
		}
	} else {
		log.Info(fmt.Sprintf("node (%s) deployment is not found", node.Name))
		log.Info(fmt.Sprintf("creating a new deployment for node (%s)", node.Name))

		if err := r.Client.Create(ctx, &newDep); err != nil {
			log.Error(err, "Unable to create node deployment")
			return err
		}
	}
	return nil
}

// createArgsForClient create arguments to be passed to the node client from node specs
func (r *NetworkReconciler) createArgsForClient(node *ethereumv1alpha1.Node, join string) []string {
	args := []string{}
	// TODO: update after admissionmutating webhook
	// because it will default all args

	if join != "" {
		args = append(args, "--network", join)
	}

	// TODO: create per client type(besu, geth ... etc)
	if node.SyncMode != "" {
		args = append(args, "--sync-mode", node.SyncMode.String())
	}

	if node.Miner {
		args = append(args, "--miner-enabled")
	}

	if node.MinerAccount != "" {
		args = append(args, "--miner-coinbase", node.MinerAccount)
	}

	if node.RPC {
		args = append(args, "--rpc-http-enabled")
	}

	if node.RPCPort != 0 {
		args = append(args, "--rpc-http-port", fmt.Sprintf("%d", node.RPCPort))
	}

	if node.RPCHost != "" {
		args = append(args, "--rpc-http-host", node.RPCHost)
	}

	if len(node.RPCAPI) != 0 {
		apis := []string{}
		for _, api := range node.RPCAPI {
			apis = append(apis, api.String())
		}
		commaSeperatedAPIs := strings.Join(apis, ",")
		args = append(args, "--rpc-http-api", commaSeperatedAPIs)
	}

	if node.WS {
		args = append(args, "--rpc-ws-enabled")
	}

	if node.WSPort != 0 {
		args = append(args, "--rpc-ws-port", fmt.Sprintf("%d", node.WSPort))
	}

	if node.WSHost != "" {
		args = append(args, "--rpc-ws-host", node.WSHost)
	}

	if len(node.WSAPI) != 0 {
		apis := []string{}
		for _, api := range node.WSAPI {
			apis = append(apis, api.String())
		}
		commaSeperatedAPIs := strings.Join(apis, ",")
		args = append(args, "--rpc-ws-api", commaSeperatedAPIs)
	}

	if len(node.Hosts) != 0 {
		commaSeperatedHosts := strings.Join(node.Hosts, ",")
		args = append(args, "--host-whitelist", commaSeperatedHosts)
	}

	if len(node.CORSDomains) != 0 {
		commaSeperatedDomains := strings.Join(node.CORSDomains, ",")
		// TODO: add graphql cors domains option if graphql is enabled
		args = append(args, "--rpc-http-cors-origins", commaSeperatedDomains)
	}

	return args
}

// createDeploymentForNode creates a new deployment for node
func (r *NetworkReconciler) createDeploymentForNode(node *ethereumv1alpha1.Node, ns string, args []string) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Name,
			Namespace: ns,
			Labels: map[string]string{
				"app": "node",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "node",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "node",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name: "node",
							// TODO: use tag
							Image: "hyperledger/besu",
							Command: []string{
								"besu",
							},
							Args: args,
						},
					},
				},
			},
		},
	}

}

// SetupWithManager adds reconciler to the manager
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ethereumv1alpha1.Network{}).
		Complete(r)
}