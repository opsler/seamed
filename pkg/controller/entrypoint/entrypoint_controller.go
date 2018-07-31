/*
Copyright 2018 Releasify.

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

package entrypoint

import (
	"context"
	"log"
	"reflect"

	seamedv1alpha1 "github.com/releasify/seamed/pkg/apis/seamed/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	knativeistio "github.com/knative/serving/pkg/apis/istio/v1alpha3"
	seamed "github.com/releasify/seamed/pkg/istio"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Entrypoint Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this seamed.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEntrypoint{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("entrypoint-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Entrypoint
	err = c.Watch(&source.Kind{Type: &seamedv1alpha1.Entrypoint{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by Entrypoint - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &seamedv1alpha1.Entrypoint{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileEntrypoint{}

// ReconcileEntrypoint reconciles a Entrypoint object
type ReconcileEntrypoint struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Entrypoint object and makes changes based on the state read
// and what is in the Entrypoint.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=seamed.releasify.io,resources=entrypoints,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileEntrypoint) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Entrypoint instance
	instance := &seamedv1alpha1.Entrypoint{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//Generate gateway from entrypoint and link them
	gateway, gatewayName := seamed.GenerateIstioGateway(*instance, instance.ObjectMeta.Namespace)
	if err := controllerutil.SetControllerReference(instance, gateway, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the Deployment already exists
	found := &knativeistio.Gateway{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: gatewayName, Namespace: gateway.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating Gateaway %s/%s\n", gateway.Namespace, gatewayName)
		err = r.Create(context.TODO(), gateway)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Update the found object and write the result back if there are any changes
	if !reflect.DeepEqual(gateway.Spec, found.Spec) {
		found.Spec = deploy.Spec
		log.Printf("Updating Gateway %s/%s\n", gateway.Namespace, gatewayName)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
