/*
Copyright 2019 The Knative Authors.
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

package knativeserving

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"knative.dev/serving-operator/version"
	mf "github.com/jcrossley3/manifestival"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	listers "knative.dev/serving-operator/pkg/client/listers/serving/v1alpha1"
	"knative.dev/serving-operator/pkg/reconciler"
	"knative.dev/serving-operator/pkg/reconciler/knativeserving/common"
)

const (
	ReconcilerName = "KnativeServing"
)

var platforms common.Platforms

type Reconciler struct {
	*reconciler.Base

	// Listers index properties about resources
	knativeServingLister          listers.KnativeServingLister
	deploymentLister              appsv1listers.DeploymentLister
	serviceLister                 corev1listers.ServiceLister
	config                        mf.Manifest
	// TODO We keep this client in for transition to accommodate the old code. It will be removed later.
	client                        client.Client
	reconcileKnativeServing       ReconcileKnativeServing
	kubeClientSet                 kubernetes.Interface
	dynamicClientSet              dynamic.Interface
	scheme                        *runtime.Scheme
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	logger := logging.FromContext(ctx)
	logger.Info("Let us do a reconcile.")

	// Get the Route resource with this namespace/name.
	original, err := c.knativeServingLister.KnativeServings(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logger.Errorf("route %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}
	// Don't modify the informers copy.
	knativeServing := original.DeepCopy()

	// Reconcile this copy of the route and then write back any status
	// updates regardless of whether the reconciliation errored out.
	reconcileErr := c.reconcile(ctx, knativeServing)
	if equality.Semantic.DeepEqual(original.Status, knativeServing.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err = c.updateStatus(knativeServing); err != nil {
		logger.Warnw("Failed to update route status", zap.Error(err))
		c.Recorder.Eventf(knativeServing, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for Route %q: %v", knativeServing.Name, err)
		return err
	}
	if reconcileErr != nil {
		c.Recorder.Event(knativeServing, corev1.EventTypeWarning, "InternalError", reconcileErr.Error())
		return reconcileErr
	}
	// TODO(mattmoor): Remove this after 0.7 cuts.
	// If the spec has changed, then assume we need an upgrade and issue a patch to trigger
	// the webhook to upgrade via defaulting.  Status updates do not trigger this due to the
	// use of the /status resource.
	if !equality.Semantic.DeepEqual(original.Spec, knativeServing.Spec) {
		routes := v1alpha1.SchemeGroupVersion.WithResource("routes")
		if err := c.MarkNeedsUpgrade(routes, knativeServing.Namespace, knativeServing.Name); err != nil {
			return err
		}
	}
	return nil
}

func (c *Reconciler) updateStatus(desired *v1alpha1.KnativeServing) (*v1alpha1.KnativeServing, error) {
	ks, err := c.knativeServingLister.KnativeServings(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}
	// If there's nothing to update, just return.
	if reflect.DeepEqual(ks.Status, desired.Status) {
		return ks, nil
	}
	// Don't modify the informers copy
	existing := ks.DeepCopy()
	existing.Status = desired.Status
	return c.KnativeServingClientSet.ServingV1alpha1().KnativeServings(desired.Namespace).UpdateStatus(existing)
}

func (c *Reconciler) reconcile(ctx context.Context, ks *v1alpha1.KnativeServing) error {
	stages := []func(context.Context, *v1alpha1.KnativeServing) error{
		c.initStatus,
		c.install,
		c.checkDeployments,
		c.deleteObsoleteResources,
	}

	for _, stage := range stages {
		if err := stage(ctx, ks); err != nil {
			return err
		}
	}
	return nil
}

// Initialize status conditions
func (r *Reconciler) initStatus(ctx context.Context, instance *v1alpha1.KnativeServing) error {
	logger := logging.FromContext(ctx)
	logger.Info("initStatus, the status is %s", instance.Status)

	if len(instance.Status.Conditions) == 0 {
		instance.Status.InitializeConditions()
		if err := r.updateStatusInit(instance); err != nil {
			return err
		}
	}
	return nil
}

// Update the status subresource
func (r *Reconciler) updateStatusInit(instance *v1alpha1.KnativeServing) error {

	// Account for https://github.com/kubernetes-sigs/controller-runtime/issues/406
	gvk := instance.GroupVersionKind()
	defer instance.SetGroupVersionKind(gvk)

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		return err
	}
	return nil
}

// Apply the embedded resources
func (r *Reconciler) install(ctx context.Context, instance *v1alpha1.KnativeServing) error {
	logger := logging.FromContext(ctx)
	logger.Info("install, the status is %s", instance.Status)
	if instance.Status.IsDeploying() {
		return nil
	}
	defer r.updateStatus(instance)

	extensions, err := platforms.Extend(r.client, r.kubeClientSet, r.dynamicClientSet, r.scheme)
	if err != nil {
		return err
	}

	err = r.config.Transform(extensions.Transform(r.scheme, instance)...)
	if err == nil {
		err = extensions.PreInstall(instance)
		if err == nil {
			err = r.config.ApplyAll()
			if err == nil {
				err = extensions.PostInstall(instance)
			}
		}
	}
	if err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}

	// Update status
	instance.Status.Version = version.Version
	logger.Info("Install succeeded, the version is %s", version.Version)
	instance.Status.MarkInstallSucceeded()
	return nil
}

// Check for all deployments available
func (r *Reconciler) checkDeployments(ctx context.Context, instance *v1alpha1.KnativeServing) error {
	logger := logging.FromContext(ctx)
	logger.Infof("Let us do a checkDeployments. the status is %s", instance.Status)
	logger.Info("The namespace for KnativeServing is %s", instance.Namespace)

	defer r.updateStatus(instance)
	available := func(d *appsv1.Deployment) bool {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}
	deployment := &appsv1.Deployment{}

	logger.Infof("how many resources: %d.", len(r.config.Resources))
	for _, u := range r.config.Resources {
		if u.GetKind() == "Deployment" {
			logger.Infof("check resource namespace is: %s.", u.GetNamespace())
			logger.Infof("check resource name is: %d.", u.GetName())
		}
	}
	for _, u := range r.config.Resources {
		if u.GetKind() == "Deployment" {
			logger.Infof("resource namespace is: %s.", u.GetNamespace())
			logger.Infof("resource name is: %d.", u.GetName())
			key := client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}
			if err := r.client.Get(ctx, key, deployment); err != nil {
				instance.Status.MarkDeploymentsNotReady()
				if errors.IsNotFound(err) {
					logger.Infof("return resource error not found")
					return nil
				}
				logger.Infof("return resource error")
				return err
			}
			if !available(deployment) {
				instance.Status.MarkDeploymentsNotReady()
				logger.Infof("return resource nil is: %s.")
				return nil
			}
		}
	}
	logger.Infof("All deployments are available")
	instance.Status.MarkDeploymentsAvailable()
	return nil
}

// Delete obsolete resources from previous versions
func (r *Reconciler) deleteObsoleteResources(ctx context.Context, instance *v1alpha1.KnativeServing) error {
	// istio-system resources from 0.3
	resource := &unstructured.Unstructured{}
	resource.SetNamespace("istio-system")
	resource.SetName("knative-ingressgateway")
	resource.SetAPIVersion("v1")
	resource.SetKind("Service")
	if err := r.config.Delete(resource); err != nil {
		return err
	}
	resource.SetAPIVersion("apps/v1")
	resource.SetKind("Deployment")
	if err := r.config.Delete(resource); err != nil {
		return err
	}
	resource.SetAPIVersion("autoscaling/v1")
	resource.SetKind("HorizontalPodAutoscaler")
	if err := r.config.Delete(resource); err != nil {
		return err
	}
	// config-controller from 0.5
	resource.SetNamespace(instance.GetNamespace())
	resource.SetName("config-controller")
	resource.SetAPIVersion("v1")
	resource.SetKind("ConfigMap")
	if err := r.config.Delete(resource); err != nil {
		return err
	}
	return nil
}
