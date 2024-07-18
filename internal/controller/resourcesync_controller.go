/*
Copyright 2024.

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

package controller

import (
	"context"
	"errors"
	"fmt"
    "time"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    meta  "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1alpha1 "github.com/EyalPazz/ResourceSyncOperator/api/v1alpha1"
)

// ResourceSyncReconciler reconciles a ResourceSync object
type ResourceSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.resourcesyncoperator.com,resources=resourcesyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.resourcesyncoperator.com,resources=resourcesyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.resourcesyncoperator.com,resources=resourcesyncs/finalizers,verbs=update
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;update;patch;
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes/custom-host,verbs=get;list;watch;update;patch;create;
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ResourceSync object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ResourceSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
    
	resourcesync := &syncv1alpha1.ResourceSync{}
	err := r.Get(ctx, req.NamespacedName, resourcesync)

	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprint("Starting Reconciliation for resource sync", resourcesync.Name))

	route := &routev1.Route{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: resourcesync.Spec.Route.Namespace,
		Name:      resourcesync.Spec.Route.Name,
	}, route)

	if err != nil {
        meta.SetStatusCondition(&resourcesync.Status.Conditions, metav1.Condition{
            Type: "Ready",
            Status: "False",
            Reason: "UnableToRetrieveRoute",
            Message: err.Error(),
        })
        r.Update(ctx, route)
		return ctrl.Result{}, err
	}

	secret := &v1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: resourcesync.Spec.Secret.Namespace,
		Name:      resourcesync.Spec.Secret.Name,
	}, secret)

	if err != nil {
        meta.SetStatusCondition(&resourcesync.Status.Conditions, metav1.Condition{
            Type: "Ready",
            Status: "False",
            Reason: "UnableToRetrieveSecret",
            Message: err.Error(),
        })
        r.Update(ctx, route)
		return ctrl.Result{}, err
	}

	secretData := make(map[string]string)

	for key, encodedValue := range secret.Data {
		secretData[key] = string(encodedValue)
	}

	if _, exists := secretData["tls.key"]; !exists {
        meta.SetStatusCondition(&resourcesync.Status.Conditions, metav1.Condition{
            Type: "Ready",
            Status: "False",
            Reason: "TLSKeyNotFound",
            Message: "tls.key field does not exist in the given secret",
        })
        r.Update(ctx, route)
		return ctrl.Result{}, errors.New("tls.key field does not exist in the given secret")
	}

	if _, exists := secretData["tls.crt"]; !exists {
        meta.SetStatusCondition(&resourcesync.Status.Conditions, metav1.Condition{
            Type: "Ready",
            Status: "False",
            Reason: "TLSCertificateNotFound",
            Message: "tls.crt field does not exist in the given secret",
        })
        r.Update(ctx, resourcesync)
		return ctrl.Result{}, errors.New("tls.crt field does not exist in the given secret")
	}

    if secretData["tls.key"] != route.Spec.TLS.Key {
        route.Spec.TLS.Key = secretData["tls.key"]
        err = r.Update(ctx, resourcesync)
        if err != nil {
            return ctrl.Result{}, err
        }

        err = r.Get(ctx, types.NamespacedName{
            Namespace: resourcesync.Spec.Route.Namespace,
            Name:      resourcesync.Spec.Route.Name,
        }, route)

        if err != nil {
            return ctrl.Result{}, err
        }

        logger.Info("Key Updated")
    }

    if secretData["tls.crt"] != route.Spec.TLS.Certificate {
        route.Spec.TLS.Certificate = secretData["tls.crt"]
        err = r.Update(ctx, resourcesync)
        if err != nil {
            return ctrl.Result{}, err
        }
        logger.Info("Certificate Updated")
    }

    if resourcesync.Status.Conditions == nil {
        resourcesync.Status.Conditions = []metav1.Condition{}
    }


    meta.SetStatusCondition(&resourcesync.Status.Conditions, metav1.Condition{
            Type: "Ready",
            Status: "True",
            Reason: "Success",
            Message: "Route successfully synced with secret",
    })

    err = r.Status().Update(ctx, resourcesync)
    if err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1alpha1.ResourceSync{}).
		Complete(r)
}
