package tfprovider

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type DataSourceBase[implType kube.StateInteraface] struct {
	Provider         *KubeProvider
	tfTypeNameSuffix string
	schema           schema.Schema
}

func (h *DataSourceBase[implType]) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	provider, ok := req.ProviderData.(*KubeProvider)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Type", "Expected provider data to be of type *KubernetesResourceProvider")
		return
	}
	h.Provider = provider
}
func (h *DataSourceBase[implType]) NewResourceHelper(ctx context.Context, state implType) (*kube.ResourceHelper, error) {
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
func (h *DataSourceBase[implType]) Read(ctx context.Context, config implType, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	resp.Diagnostics.Append(req.Config.Get(ctx, config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceBase, err := h.NewResourceHelper(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource helper", err.Error())
		return
	}
	diags := h.Fetch(ctx, resourceBase, config, kube.MayOrMayNotExist)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}

func (h *DataSourceBase[implType]) Fetch(ctx context.Context, resourceBase *kube.ResourceHelper, config implType, existRequirement kube.ExistRequirement) diag.Diagnostics {
	var diags diag.Diagnostics
	fetch := GetPtrToEmbedddedType[tfparts.FetchMap](config)
	compiledFetch, err := fetch.Compile()
	if err != nil {
		diags.AddError("Failed to compile fetch map", err.Error())
		return diags
	}
	outputs, err := resourceBase.Fetch(ctx, config, compiledFetch, existRequirement)
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
func (h *DataSourceBase[implType]) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	if h.Provider == nil {
		resp.TypeName = req.ProviderTypeName + h.tfTypeNameSuffix
		return
	}
	resp.TypeName = h.Provider.typeName + h.tfTypeNameSuffix
}
func (h *DataSourceBase[implType]) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = h.schema
}
