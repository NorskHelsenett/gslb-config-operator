/*
Copyright 2026.

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

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	lbv1alpha1 "github.com/NorskHelsenett/gslb-config-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var gslbservicelog = logf.Log.WithName("gslbservice-resource")

// SetupGSLBServiceWebhookWithManager registers the webhook for GSLBService in the manager.
func SetupGSLBServiceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &lbv1alpha1.GSLBService{}).
		WithValidator(&GSLBServiceCustomValidator{}).
		WithDefaulter(&GSLBServiceCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-lb-nhn-no-v1alpha1-gslbservice,mutating=true,failurePolicy=fail,sideEffects=None,groups=lb.nhn.no,resources=gslbservices,verbs=create;update,versions=v1alpha1,name=mgslbservice-v1alpha1.kb.io,admissionReviewVersions=v1

// GSLBServiceCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind GSLBService when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type GSLBServiceCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind GSLBService.
func (d *GSLBServiceCustomDefaulter) Default(_ context.Context, obj *lbv1alpha1.GSLBService) error {
	gslbservicelog.Info("Defaulting for GSLBService", "name", obj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-lb-nhn-no-v1alpha1-gslbservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=lb.nhn.no,resources=gslbservices,verbs=create;update,versions=v1alpha1,name=vgslbservice-v1alpha1.kb.io,admissionReviewVersions=v1

// GSLBServiceCustomValidator struct is responsible for validating the GSLBService resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GSLBServiceCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GSLBService.
func (v *GSLBServiceCustomValidator) ValidateCreate(_ context.Context, obj *lbv1alpha1.GSLBService) (admission.Warnings, error) {
	gslbservicelog.Info("Validation for GSLBService upon creation", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GSLBService.
func (v *GSLBServiceCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *lbv1alpha1.GSLBService) (admission.Warnings, error) {
	gslbservicelog.Info("Validation for GSLBService upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GSLBService.
func (v *GSLBServiceCustomValidator) ValidateDelete(_ context.Context, obj *lbv1alpha1.GSLBService) (admission.Warnings, error) {
	gslbservicelog.Info("Validation for GSLBService upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
