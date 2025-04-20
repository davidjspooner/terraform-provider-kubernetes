package tfprovider

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type ResourceBase[implType kresource.StateInteraface] struct {
	Provider *KubernetesResourceProvider
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
func (h *ResourceBase[implType]) Create(ctx context.Context, plan implType, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options, err := kresource.MergeAPIOptions(h.Provider.DefaultApiOptions, plan.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Failed to merge API options", err.Error())
		return
	}
	resourceBase := kresource.NewResourceHelper(&h.Provider.Shared, options)
	err = resourceBase.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (h *ResourceBase[implType]) Read(ctx context.Context, plan, state implType, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.Append(req.State.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options, err := kresource.MergeAPIOptions(h.Provider.DefaultApiOptions, plan.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Failed to merge API options", err.Error())
		return
	}
	resourceBase := kresource.NewResourceHelper(&h.Provider.Shared, options)
	err = resourceBase.Read(ctx, plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (h *ResourceBase[implType]) Update(ctx context.Context, plan implType, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options, err := kresource.MergeAPIOptions(h.Provider.DefaultApiOptions, plan.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Failed to merge API options", err.Error())
		return
	}
	resourceBase := kresource.NewResourceHelper(&h.Provider.Shared, options)
	err = resourceBase.Update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
func (h *ResourceBase[implType]) Delete(ctx context.Context, state implType, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options, err := kresource.MergeAPIOptions(h.Provider.DefaultApiOptions, state.GetApiOptions())
	if err != nil {
		resp.Diagnostics.AddError("Failed to merge API options", err.Error())
		return
	}
	resourceBase := kresource.NewResourceHelper(&h.Provider.Shared, options)
	err = resourceBase.Delete(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}
