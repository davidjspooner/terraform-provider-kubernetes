package tfprovider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesWatch{}

func init() {
	// Register the resource with the provider.
	RegisterResource(func() resource.Resource {
		r := &KubernetesWatch{
			tfTypeNameSuffix: "_watch",
		}
		return r
	})
}

// KubernetesWatch defines the resource implementation.
type KubernetesWatch struct {
	ResourceBase[*WatchModel]
	tfTypeNameSuffix string
}

// WatchModel describes the resource data model.
type WatchModel struct {
	ApiVersion  types.String             `tfsdk:"api_version"`
	Kind        types.String             `tfsdk:"kind"`
	Metadata    *tfparts.ShortMetadata   `tfsdk:"metadata"`
	State       types.String             `tfsdk:"state"`
	LastChecked types.String             `tfsdk:"last_checked"`
	ApiOptions  *tfparts.APIOptionsModel `tfsdk:"api_options"`
	tfparts.FetchMap
}

func (model *WatchModel) BuildManifest(manifest *unstructured.Unstructured) error {
	// We don't need to build a manifest for this resource
	return nil
}

func (model *WatchModel) UpdateFrom(manifest unstructured.Unstructured) error {
	return nil
}

func (model *WatchModel) GetResouceKey() (kresource.ResourceKey, error) {
	if model.Metadata == nil {
		return kresource.ResourceKey{}, fmt.Errorf("metadata is nil")
	}
	if model.Metadata.Name.IsNull() {
		return kresource.ResourceKey{}, fmt.Errorf("name is nil")
	}
	if model.Kind.IsNull() {
		return kresource.ResourceKey{}, fmt.Errorf("kind is nil")
	}
	if model.ApiVersion.IsNull() {
		return kresource.ResourceKey{}, fmt.Errorf("api_version is nil")
	}
	var namespace *string
	if model.Metadata.Namespace.IsNull() {
		namespace = nil
	} else {
		s := model.Metadata.Namespace.ValueString()
		namespace = &s
	}
	k := kresource.ResourceKey{
		Kind:       model.Kind.ValueString(),
		ApiVersion: model.ApiVersion.ValueString(),
	}
	k.MetaData.Name = model.Metadata.Name.ValueString()
	k.MetaData.Namespace = namespace
	return k, nil
}

func (r *KubernetesWatch) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesWatch) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attr := map[string]schema.Attribute{
		"state": schema.StringAttribute{
			MarkdownDescription: "used to trick Terraform into thinking the resource has changed",
			Computed:            true,
			Default: &stringValueFunc{func() string {
				return "checked"
			}, "used to trick Terraform into thinking the resource has changed"},
		},
		"last_checked": schema.StringAttribute{
			MarkdownDescription: "Last time the resource was checked",
			Computed:            true,
		},
		"kind": schema.StringAttribute{
			MarkdownDescription: "Kind of the object",
			Required:            true,
		},
	}
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Wait until a resource attribute matches a regex",

		Attributes: MergeResourceAttributes(
			attr,
			tfparts.FetchRequestAttributes(),
			tfparts.ApiOptionsResourceAttributes(),
		),
		Blocks: map[string]schema.Block{
			"metadata": tfparts.ShortMetadataSchemaBlock(),
		},
	}
}

func (r *KubernetesWatch) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	r.ResourceBase.Configure(ctx, req, resp)
}

func (r *KubernetesWatch) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WatchModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesWatch) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WatchModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := errors.New("TODO: implement watch")

	if err != nil {
		state.State = types.StringValue("stale")
	} else {
		var subDiags diag.Diagnostics
		state.Output = basetypes.NewMapNull(types.StringType)
		resp.Diagnostics.Append(subDiags...)
	}

	state.LastChecked = types.StringValue(time.Now().Format(time.RFC3339Nano))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KubernetesWatch) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WatchModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesWatch) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	//effectily a no-op - we just reset the state vars
	resp.State.RemoveResource(ctx)
}
