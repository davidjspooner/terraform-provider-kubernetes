package tfprovider

import (
	"context"
	"errors"
	"reflect"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StateInteraface interface {
	GetResouceKey() (kresource.Key, error)
	BuildManifest(manifest *unstructured.Unstructured) error
	FromManifest(manifest *unstructured.Unstructured) error
	GetApiOptions() *APIOptions
}

type BaseResourceHandler[implType StateInteraface] struct {
	provider *KubernetesResourceProvider
}

func NewCommonHandler[implType StateInteraface](ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) *BaseResourceHandler[implType] {
	provider, ok := req.ProviderData.(*KubernetesResourceProvider)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected KubernetesProvider")
		return nil
	}
	handler := &BaseResourceHandler[implType]{
		provider: provider,
	}
	return handler
}

func (base *BaseResourceHandler[implType]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan implType
	plan = reflect.New(reflect.TypeOf(plan).Elem()).Interface().(implType)

	req.Plan.Get(ctx, plan)
	apiOptions, err := MergeKubenetesAPIOptions(base.provider.DefaultApiOptions, plan.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Error determining api options", err.Error())
		return
	}
	retryHelper, err := apiOptions.Retry.NewHelper()
	if err != nil {
		resp.Diagnostics.AddError("Error creating retry helper", err.Error())
		return
	}

	var manifest unstructured.Unstructured
	err = plan.BuildManifest(&manifest)
	if err != nil {
		resp.Diagnostics.AddError("Error building manifest", err.Error())
		return
	}

	key, err := plan.GetResouceKey()
	if err != nil {
		resp.Diagnostics.AddError("Error getting resource key", err.Error())
		return
	}

	alreadyExists := false
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := base.provider.Shared.Apply(ctx, &key, manifest, apiOptions)
		if apierrors.IsAlreadyExists(err) {
			alreadyExists = true
			return nil
		}
		return err
	})
	if alreadyExists {
		resp.Diagnostics.AddError("Resource already exists", "Resource already exists")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	err = plan.FromManifest(&manifest)
	if err != nil {
		resp.Diagnostics.AddError("Error setting from actual", err.Error())
		return
	}
	resp.State.Set(ctx, &plan)
}

func (base *BaseResourceHandler[implType]) trimMapElements(previousMap, currentMap map[string]string) {
	deleteKeys := make([]string, 0, len(previousMap))
	for k := range currentMap {
		_, existed := previousMap[k]
		if !existed {
			deleteKeys = append(deleteKeys, k)
		}
	}
	for _, deleteKey := range deleteKeys {
		delete(currentMap, deleteKey)
	}
}

func (base *BaseResourceHandler[implType]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var plan, state implType
	plan = reflect.New(reflect.TypeOf(plan).Elem()).Interface().(implType)
	state = reflect.New(reflect.TypeOf(state).Elem()).Interface().(implType)

	req.State.Get(ctx, plan)
	req.State.Get(ctx, state)
	apiOptions, err := MergeKubenetesAPIOptions(base.provider.DefaultApiOptions, plan.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Error determining api options", err.Error())
		return
	}
	retryHelper, err := apiOptions.Retry.NewHelper()
	if err != nil {
		resp.Diagnostics.AddError("Error creating retry helper", err.Error())
		return
	}

	key, err := state.GetResouceKey()
	if err != nil {
		resp.Diagnostics.AddError("Error getting resource key", err.Error())
		return
	}

	var currentStateManifest unstructured.Unstructured
	found := false
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		var err error
		currentStateManifest, err = base.provider.Shared.Get(ctx, &key, apiOptions)
		if err == nil {
			found = true
			return nil
		}
		if apierrors.IsNotFound(err) {
			// resource not found, return nil
			return nil
		}
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading current state", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	var previousStateManifest unstructured.Unstructured
	state.BuildManifest(&previousStateManifest)

	currentStateAnnotations := currentStateManifest.GetAnnotations()
	previousStateAnnotations := previousStateManifest.GetAnnotations()
	if previousStateAnnotations == nil {
		currentStateManifest.SetAnnotations(nil)
	} else if currentStateAnnotations != nil {
		base.trimMapElements(previousStateAnnotations, currentStateAnnotations)
		currentStateManifest.SetAnnotations(currentStateAnnotations)
	}
	currentStateLabels := currentStateManifest.GetLabels()
	previousStateLabels := previousStateManifest.GetLabels()
	if previousStateLabels == nil {
		currentStateManifest.SetLabels(nil)
	} else if currentStateLabels != nil {
		base.trimMapElements(previousStateLabels, currentStateLabels)
		currentStateManifest.SetLabels(currentStateLabels)
	}

	err = state.FromManifest(&currentStateManifest)

	if err != nil {
		resp.Diagnostics.AddError("Error setting from actual", err.Error())
		return
	}
	resp.State.Set(ctx, &state)
}
func (base *BaseResourceHandler[implType]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan implType
	plan = reflect.New(reflect.TypeOf(plan).Elem()).Interface().(implType)
	req.Plan.Get(ctx, &plan)

	apiOptions, err := MergeKubenetesAPIOptions(base.provider.DefaultApiOptions, plan.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Error determining api options", err.Error())
		return
	}
	retryHelper, err := apiOptions.Retry.NewHelper()
	if err != nil {
		resp.Diagnostics.AddError("Error creating retry helper", err.Error())
		return
	}

	var manifest unstructured.Unstructured
	err = plan.BuildManifest(&manifest)
	if err != nil {
		resp.Diagnostics.AddError("Error building manifest", err.Error())
		return
	}
	key, err := plan.GetResouceKey()
	if err != nil {
		resp.Diagnostics.AddError("Error getting resource key", err.Error())
		return
	}
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := base.provider.Shared.Apply(ctx, &key, manifest, apiOptions)
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}
	manifest, err = base.provider.Shared.Get(ctx, &key, apiOptions)
	if err != nil {
		resp.Diagnostics.AddError("Error reading current state", err.Error())
		return
	}
	err = plan.FromManifest(&manifest)
	if err != nil {
		resp.Diagnostics.AddError("Error setting from actual", err.Error())
		return
	}
	resp.State.Set(ctx, plan)
}
func (base *BaseResourceHandler[implType]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state implType
	state = reflect.New(reflect.TypeOf(state).Elem()).Interface().(implType)
	req.State.Get(ctx, &state)
	apiOptions, err := MergeKubenetesAPIOptions(base.provider.DefaultApiOptions, state.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Error determining api options", err.Error())
		return
	}
	retryHelper, err := apiOptions.Retry.NewHelper()
	if err != nil {
		resp.Diagnostics.AddError("Error creating retry helper", err.Error())
		return
	}

	key, err := state.GetResouceKey()
	if err != nil {
		resp.Diagnostics.AddError("Error getting resource key", err.Error())
		return
	}
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		base.provider.Shared.Delete(ctx, &key, apiOptions)
		_, err = base.provider.Shared.Get(ctx, &key, apiOptions)

		if err != nil && apierrors.IsNotFound(err) {
			//excellent
			return nil
		}
		return errors.New("resource still exists")
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}
