/*
Copyright 2019 The cert-manager authors.
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

package certmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apiutil "github.com/jetstack/cert-manager/pkg/api/util"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certmanagerv1beta1 "github.com/sapcc/digicert-issuer/apis/certmanager/v1beta1"
	"github.com/sapcc/digicert-issuer/pkg/provisioners"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificateRequestReconciler reconciles a DigicertIssuer object.
type CertificateRequestReconciler struct {
	client.Client
	Log                                logr.Logger
	Scheme                             *runtime.Scheme
	BackoffDurationProvisionerNotReady time.Duration
	recorder                           record.EventRecorder
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=get;update;patch

// Reconcile will read and validate a DigicertIssuer resource associated to the
// CertificateRequest resource, and it will sign the CertificateRequest with the
// provisioner in the DigicertIssuer.
func (r *CertificateRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("certificaterequest", req.NamespacedName)

	// Fetch the CertificateRequest resource being reconciled.
	// Just ignore the request if the certificate request has been deleted.
	cr := new(cmapi.CertificateRequest)
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Error(err, "failed to retrieve CertificateRequest resource")
		return ctrl.Result{}, err
	}

	// Check the CertificateRequest's issuerRef and if it does not match the api
	// group name, log a message at a debug level and stop processing.
	if cr.Spec.IssuerRef.Group != "" && cr.Spec.IssuerRef.Group != certmanagerv1beta1.GroupVersion.Group {
		log.V(4).Info("resource does not specify an issuerRef group name that we are responsible for", "group", cr.Spec.IssuerRef.Group)
		return ctrl.Result{}, nil
	}

	// If the certificate data is already set then we skip this request as it
	// has already been completed in the past.
	if len(cr.Status.Certificate) > 0 {
		log.V(4).Info("existing certificate data found in status, skipping already completed CertificateRequest")
		return ctrl.Result{}, nil
	}

	iss := new(certmanagerv1beta1.DigicertIssuer)
	issNamespaceName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      cr.Spec.IssuerRef.Name,
	}
	if err := r.Client.Get(ctx, issNamespaceName, iss); err != nil {
		log.Error(err, "failed to retrieve DigicertIssuer resource", "namespace", req.Namespace, "name", cr.Spec.IssuerRef.Name)
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "Failed to retrieve DigicertIssuer resource %s: %v", issNamespaceName, err)
		return ctrl.Result{}, err
	}

	if !isDigicertIssuerReady(iss) {
		err := fmt.Errorf("resource %s is not ready", issNamespaceName)
		log.Error(err, "issuers is not ready")
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "DigicertIssuer resource %s is not Ready", issNamespaceName)
		return ctrl.Result{Requeue: true, RequeueAfter: r.BackoffDurationProvisionerNotReady}, err
	}

	// Load the provisioner that will sign the CertificateRequest.
	provisioner, ok := provisioners.Load(issNamespaceName)
	if !ok {
		err := fmt.Errorf("provisioner %s not found", issNamespaceName)
		log.Error(err, "failed to load provisioner for DigicertIssuer resource")
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "Failed to load provisioner for DigicertIssuer resource %s", issNamespaceName)
		return ctrl.Result{Requeue: true, RequeueAfter: r.BackoffDurationProvisionerNotReady}, err
	}

	// Sign CertificateRequest.
	signedPEM, err := provisioner.Sign(ctx, cr)
	if err != nil {
		log.Error(err, "failed to sign certificate request")
		return ctrl.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonFailed, "Failed to sign certificate request: %v", err)
	}
	cr.Status.Certificate = signedPEM

	err = r.setStatus(ctx, cr, cmmeta.ConditionTrue, cmapi.CertificateRequestReasonIssued, "Certificate issued")
	return ctrl.Result{}, err
}

// SetupWithManager initializes the CertificateRequest controller into the
// controller runtime.
func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("certificateRequestController")
	return ctrl.NewControllerManagedBy(mgr).
		For(&cmapi.CertificateRequest{}).
		Complete(r)
}

func isDigicertIssuerReady(issuer *certmanagerv1beta1.DigicertIssuer) bool {
	status := issuer.Status
	if status == nil {
		return false
	}

	for _, condition := range status.Conditions {
		if condition.Type == certmanagerv1beta1.ConditionReady && condition.Status == certmanagerv1beta1.ConditionTrue {
			return true
		}
	}

	return false
}

func (r *CertificateRequestReconciler) setStatus(ctx context.Context, cr *cmapi.CertificateRequest, status cmmeta.ConditionStatus, reason, message string, args ...interface{}) error {
	completeMessage := fmt.Sprintf(message, args...)
	apiutil.SetCertificateRequestCondition(cr, cmapi.CertificateRequestConditionReady, status, reason, completeMessage)

	// Fire an Event to additionally inform users of the change
	eventType := core.EventTypeNormal
	if status == cmmeta.ConditionFalse {
		eventType = core.EventTypeWarning
	}
	r.recorder.Event(cr, eventType, reason, completeMessage)

	return r.Client.Status().Update(ctx, cr)
}
