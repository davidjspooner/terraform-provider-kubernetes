package tfprovider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceKubeContext{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		ds := &DataSourceKubeContext{
			tfTypeNameSuffix: "_context",
		}
		return ds
	})
}

// DataSourceKubeFiles defines the resource implementation.
type DataSourceKubeContext struct {
	tfTypeNameSuffix string
	Provider         *KubeProvider
}

// CurrentContextModel describes the resource data tfshared.
type CurrentContextModel struct {
	User      types.String `tfsdk:"user"`
	Host      types.String `tfsdk:"host"`
	Cluster   types.String `tfsdk:"cluster"`
	Namespace types.String `tfsdk:"namespace"`
}

func (r *DataSourceKubeContext) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}
func (r *DataSourceKubeContext) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attr := map[string]schema.Attribute{
		"user": schema.StringAttribute{
			MarkdownDescription: "The user name for the current context",
			Computed:            true,
		},
		"host": schema.StringAttribute{
			MarkdownDescription: "The host for the current context",
			Computed:            true,
		},
		"cluster": schema.StringAttribute{
			MarkdownDescription: "The cluster for the current context",
			Computed:            true,
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "The namespace for the current context",
			Computed:            true,
		},
	}
	resp.Schema = schema.Schema{
		Description: "Get the current context from the kubeconfig file",
		Attributes:  MergeDataAttributes(attr),
	}
}

func (d *DataSourceKubeContext) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	provider, ok := req.ProviderData.(*KubeProvider)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Type", "Expected provider data to be of type *KubernetesResourceProvider")
		return
	}
	d.Provider = provider
}

func (r *DataSourceKubeContext) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if r.Provider == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider is not configured or is missing required data.")
		return
	}

	currentContext, err := r.Provider.Shared.GetCurrentContext()
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch current context", err.Error())
		return
	}

	state := CurrentContextModel{
		User:      types.StringValue(currentContext.User),
		Host:      types.StringValue(currentContext.Host),
		Cluster:   types.StringValue(currentContext.Cluster),
		Namespace: types.StringValue(currentContext.Namespace),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
