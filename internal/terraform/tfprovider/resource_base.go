package tfprovider

import (
	"context"
	"reflect"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ResourceBase[implType kube.StateInteraface] struct {
	Provider         *KubernetesResourceProvider
	tfTypeNameSuffix string
	schema           schema.Schema
}

func (h *ResourceBase[implType]) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	provider, ok := req.ProviderData.(*KubernetesResourceProvider)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Type", "Expected provider data to be of type *KubernetesResourceProvider")
		return
	}
	h.Provider = provider
}
func (h *ResourceBase[implType]) NewResourceHelper(ctx context.Context, state implType) (*kube.ResourceHelper, error) {

	resourceOptions := GetPtrToEmbedddedType[tfparts.APIOptionsModel](state)
	options, err := kube.MergeAPIOptions(h.Provider.DefaultApiOptions, resourceOptions.Options())
	if err != nil {
		return nil, err
	}
	key, err := state.GetResouceKey()
	if err != nil {
		return nil, err
	}
	resourceBase, err := kube.NewResourceHelper(ctx, &h.Provider.Shared, options, key)
	if err != nil {
		return nil, err
	}
	return resourceBase, nil
}

func (h *ResourceBase[implType]) Create(ctx context.Context, plan implType, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceBase, err := h.NewResourceHelper(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource helper", err.Error())
		return
	}

	err = resourceBase.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}
	diags := h.Fetch(ctx, resourceBase, plan, kube.MustExit)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (h *ResourceBase[implType]) Read(ctx context.Context, state implType, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceBase, err := h.NewResourceHelper(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource helper", err.Error())
		return
	}

	diags := h.Fetch(ctx, resourceBase, state, kube.MayOrMayNotExist)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (h *ResourceBase[implType]) Fetch(ctx context.Context, resourceBase *kube.ResourceHelper, state implType, existRequirement kube.ExistRequirement) diag.Diagnostics {
	var diags diag.Diagnostics
	fetch := GetPtrToEmbedddedType[tfparts.FetchMap](state)
	compiledFetch, err := fetch.Compile()
	if err != nil {
		diags.AddError("Failed to compile fetch map", err.Error())
		return diags
	}
	outputs, err := resourceBase.Fetch(ctx, state, compiledFetch, existRequirement)
	if err != nil {
		diags.AddError("Fetch failed", err.Error())
		return diags
	}
	if outputs != nil {
		m := make(map[string]attr.Value)
		for k, v := range outputs {
			m[k] = basetypes.NewStringValue(v)
		}
		var diags diag.Diagnostics
		fetch.Output, diags = basetypes.NewMapValue(types.StringType, m)
		if diags.HasError() {
			diags.Append(diags...)
			return diags
		}
	}
	return diags
}

func (h *ResourceBase[implType]) Update(ctx context.Context, plan implType, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceBase, err := h.NewResourceHelper(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource helper", err.Error())
		return
	}
	err = resourceBase.Update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}
	diags := h.Fetch(ctx, resourceBase, plan, kube.MustExit)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
func (h *ResourceBase[implType]) Delete(ctx context.Context, state implType, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceBase, err := h.NewResourceHelper(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource helper", err.Error())
		return
	}
	err = resourceBase.Delete(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func GetPtrToEmbedddedType[T any](iStruct any) *T {
	rValue := reflect.ValueOf(iStruct)
	if rValue.Kind() != reflect.Ptr {
		return nil
	}
	for rValue.Kind() == reflect.Ptr || rValue.Kind() == reflect.Interface {
		if rValue.IsNil() {
			return nil
		}
		rValue = rValue.Elem()
	}
	if rValue.Kind() != reflect.Struct {
		return nil
	}

	findType := reflect.TypeOf((*T)(nil)).Elem()

	rType := rValue.Type()
	for i := 0; i < rType.NumField(); i++ {
		field := rType.Field(i)
		if field.Type == findType {
			vSub := rValue.Field(i)
			return vSub.Addr().Interface().(*T)
		}
		if field.Type.Kind() == reflect.Ptr && field.Type.Elem() == findType {
			vSub := rValue.Field(i)
			if vSub.IsNil() {
				return nil
			}
			return vSub.Interface().(*T)
		}
	}
	return nil
}
