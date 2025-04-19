/*
MIT License

Copyright (c) 2020-2024 1Password

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"fmt"
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/1Password/connect-sdk-go/connect"

	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	"github.com/1Password/onepassword-operator/pkg/logs"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	"github.com/1Password/onepassword-operator/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logDaemonSet = logf.Log.WithName("controller_daemonset")

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	OpConnectClient    connect.Client
	OpAnnotationRegExp *regexp.Regexp
}

//+kubebuilder:rbac:groups=apps,resources=daemonSets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonSets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=daemonSets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OnePasswordItem object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logDaemonSet.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.V(logs.DebugLevel).Info("Reconciling DaemonSet")

	daemonSet := &appsv1.DaemonSet{}
	err := r.Get(context.Background(), req.NamespacedName, daemonSet)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	annotations, annotationsFound := op.GetAnnotationsForDaemonSet(daemonSet, r.OpAnnotationRegExp)
	if !annotationsFound {
		reqLogger.V(logs.DebugLevel).Info("No 1Password Annotations found")
		return ctrl.Result{}, nil
	}

	//If the daemonSet is not being deleted
	if daemonSet.ObjectMeta.DeletionTimestamp.IsZero() {
		// Adds a finalizer to the daemonSet if one does not exist.
		// This is so we can handle cleanup of associated secrets properly
		if !utils.ContainsString(daemonSet.ObjectMeta.Finalizers, finalizer) {
			daemonSet.ObjectMeta.Finalizers = append(daemonSet.ObjectMeta.Finalizers, finalizer)
			if err = r.Update(context.Background(), daemonSet); err != nil {
				return reconcile.Result{}, err
			}
		}
		// Handles creation or updating secrets for daemonSet if needed
		if err = r.handleApplyingDaemonSet(daemonSet, daemonSet.Namespace, annotations, req); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// The daemonSet has been marked for deletion. If the one password
	// finalizer is found there are cleanup tasks to perform
	if utils.ContainsString(daemonSet.ObjectMeta.Finalizers, finalizer) {

		secretName := annotations[op.NameAnnotation]
		if err = r.cleanupKubernetesSecretForDaemonSet(secretName, daemonSet); err != nil {
			return ctrl.Result{}, err
		}

		// Remove the finalizer from the daemonSet so deletion of daemonSet can be completed
		if err = r.removeOnePasswordFinalizerFromDaemonSet(daemonSet); err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		Complete(r)
}

func (r *DaemonSetReconciler) cleanupKubernetesSecretForDaemonSet(secretName string, deletedDaemonSet *appsv1.DaemonSet) error {
	kubernetesSecret := &corev1.Secret{}
	kubernetesSecret.ObjectMeta.Name = secretName
	kubernetesSecret.ObjectMeta.Namespace = deletedDaemonSet.Namespace

	if len(secretName) == 0 {
		return nil
	}
	updatedSecrets := map[string]*corev1.Secret{secretName: kubernetesSecret}

	multipleDaemonSetsUsingSecret, err := r.areMultipleDaemonSetsUsingSecret(updatedSecrets, *deletedDaemonSet)
	if err != nil {
		return err
	}

	// Only delete the associated kubernetes secret if it is not being used by other daemonSets
	if !multipleDaemonSetsUsingSecret {
		if err = r.Delete(context.Background(), kubernetesSecret); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (r *DaemonSetReconciler) areMultipleDaemonSetsUsingSecret(updatedSecrets map[string]*corev1.Secret, deletedDaemonSet appsv1.DaemonSet) (bool, error) {
	daemonSets := &appsv1.DaemonSetList{}
	opts := []client.ListOption{
		client.InNamespace(deletedDaemonSet.Namespace),
	}

	err := r.List(context.Background(), daemonSets, opts...)
	if err != nil {
		logDaemonSet.Error(err, "Failed to list kubernetes DaemonSets")
		return false, err
	}

	for i := 0; i < len(daemonSets.Items); i++ {
		if daemonSets.Items[i].Name != deletedDaemonSet.Name {
			if op.IsDaemonSetUsingSecrets(&daemonSets.Items[i], updatedSecrets) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *DaemonSetReconciler) removeOnePasswordFinalizerFromDaemonSet(daemonSet *appsv1.DaemonSet) error {
	daemonSet.ObjectMeta.Finalizers = utils.RemoveString(daemonSet.ObjectMeta.Finalizers, finalizer)
	return r.Update(context.Background(), daemonSet)
}

func (r *DaemonSetReconciler) handleApplyingDaemonSet(daemonSet *appsv1.DaemonSet, namespace string, annotations map[string]string, request reconcile.Request) error {
	reqLog := logDaemonSet.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	secretName := annotations[op.NameAnnotation]
	secretLabels := map[string]string(nil)
	secretType := string(corev1.SecretTypeOpaque)

	if len(secretName) == 0 {
		reqLog.Info("No 'item-name' annotation set. 'item-path' and 'item-name' must be set as annotations to add new secret.")
		return nil
	}

	item, err := op.GetOnePasswordItemByPath(r.OpConnectClient, annotations[op.ItemPathAnnotation])
	if err != nil {
		return fmt.Errorf("Failed to retrieve item: %v", err)
	}

	// Create owner reference.
	gvk, err := apiutil.GVKForObject(daemonSet, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not to retrieve group version kind: %v", err)
	}
	ownerRef := &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       daemonSet.GetName(),
		UID:        daemonSet.GetUID(),
	}

	return kubeSecrets.CreateKubernetesSecretFromItem(r.Client, secretName, namespace, item, annotations[op.AutoRestartWorkloadAnnotation], secretLabels, secretType, ownerRef)
}
