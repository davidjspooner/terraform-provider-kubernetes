// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"
	"fmt"
	"time"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesWatch{}

func init() {
	// Register the resource with the provider.
	tfprovider.RegisterResource(func() resource.Resource {
		r := &KubernetesWatch{
			tfTypeNameSuffix: "_watch",
		}
		return r
	})
}

// KubernetesWatch defines the resource implementation.
type KubernetesWatch struct {
	provider         *tfprovider.KubernetesResourceProvider
	tfTypeNameSuffix string
}

// WatchModel describes the resource data model.
type WatchModel struct {
	ApiVersion types.String              `tfsdk:"api_version"`
	Kind       types.String              `tfsdk:"kind"`
	Metadata   *tfprovider.ShortMetadata `tfsdk:"metadata"`

	Querry tfprovider.QuerryList `tfsdk:"querry"`

	Captured types.Map `tfsdk:"captured"`

	State       types.String `tfsdk:"state"`
	LastChecked types.String `tfsdk:"last_checked"`

	ApiOptions *tfprovider.APIOptionsModel `tfsdk:"api_options"`

	tfprovider.OutputMetadata
}

func (r *KubernetesWatch) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesWatch) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Wait until a resource attribute matches a regex",

		Attributes: map[string]schema.Attribute{
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
			"api_version": schema.StringAttribute{
				MarkdownDescription: "API version of the object",
				Required:            true,
			},
			"retry":  job.DefineRetryModelSchema(),
			"querry": tfprovider.QuerrySchemaList(true),
			"captured": schema.MapAttribute{
				MarkdownDescription: "Captured data",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"resource_version": schema.StringAttribute{
				MarkdownDescription: "The resource version.",
				Computed:            true,
			},
			"uid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the resource.",
				Computed:            true,
			},
			"generation": schema.Int64Attribute{
				MarkdownDescription: "The generation of the resource.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": tfprovider.ShortMetadataSchemaBlock(),
		},
	}
}

func (r *KubernetesWatch) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.provider, ok = req.ProviderData.(*tfprovider.KubernetesResourceProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubernetesProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
}

func (r *KubernetesWatch) checvaluesOnce(ctx context.Context, key *kresource.Key, data *WatchModel) (*tfprovider.CaptureMap, error) {

	apiOptions, err := kresource.MergeAPIOptions(r.provider.DefaultApiOptions, data.ApiOptions.Options())
	if err != nil {
		return nil, err
	}

	var captureMap tfprovider.CaptureMap
	unstructured, err := r.provider.Shared.Get(ctx, key, apiOptions)
	if err != nil {
		return nil, err
	}

	querryList := &data.Querry
	captureMap, err = querryList.Check(unstructured.Object)
	if err != nil {
		return nil, err
	}
	if len(captureMap) == 0 {
		return nil, fmt.Errorf("no querry match")
	}
	return &captureMap, nil
}

func (r *KubernetesWatch) checkValuesWithRetry(ctx context.Context, key *kresource.Key, data *WatchModel, diags *diag.Diagnostics) {

	var savedCaptureMap *tfprovider.CaptureMap

	err := r.retry(ctx, data, func(ctx context.Context, attempt int) error {
		var innerErr error
		savedCaptureMap, innerErr = r.checvaluesOnce(ctx, key, data)
		if innerErr != nil {
			return innerErr
		}
		return nil
	})

	if err != nil {
		diags.AddError("Failed to fetch and check", err.Error())
	}

	var subDiags diag.Diagnostics
	data.Captured, subDiags = types.MapValueFrom(ctx, types.StringType, *savedCaptureMap)
	diags.Append(subDiags...)

	data.LastChecked = types.StringValue(time.Now().Format(time.RFC3339Nano))
}

func (r *KubernetesWatch) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WatchModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var namespace string
	if data.Metadata.Namespace.IsNull() {
		namespace = r.provider.Shared.GetNamespace(nil)
	} else {
		namespace = data.Metadata.Namespace.ValueString()
	}

	key := kresource.Key{
		Kind:       data.Kind.ValueString(),
		ApiVersion: data.ApiVersion.ValueString(),
		MetaData: kresource.MetaData{
			Namespace: &namespace,
			Name:      data.Metadata.Name.ValueString(),
		},
	}

	r.checkValuesWithRetry(ctx, &key, &data, &resp.Diagnostics)

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

	var namespace string
	if state.Metadata.Namespace.IsNull() {
		namespace = r.provider.Shared.GetNamespace(nil)
	} else {
		namespace = state.Metadata.Namespace.ValueString()
	}

	key := kresource.Key{
		Kind:       state.Kind.ValueString(),
		ApiVersion: state.ApiVersion.ValueString(),
		MetaData: kresource.MetaData{
			Namespace: &namespace,
			Name:      state.Metadata.Name.ValueString(),
		},
	}

	savedCaptureMap, err := r.checvaluesOnce(ctx, &key, &state)

	if err != nil {
		state.State = types.StringValue("stale")
	} else {
		var subDiags diag.Diagnostics
		state.Captured, subDiags = types.MapValueFrom(ctx, types.StringType, *savedCaptureMap)
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

	var namespace string
	if data.Metadata.Namespace.IsNull() {
		namespace = r.provider.Shared.GetNamespace(nil)
	} else {
		namespace = data.Metadata.Namespace.ValueString()
	}
	key := kresource.Key{
		Kind:       data.Kind.ValueString(),
		ApiVersion: data.ApiVersion.ValueString(),
		MetaData: kresource.MetaData{
			Name:      data.Metadata.Name.ValueString(),
			Namespace: &namespace,
		},
	}

	r.checkValuesWithRetry(ctx, &key, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesWatch) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	//effectily a no-op - we just reset the output vars

	var data WatchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var subDiags diag.Diagnostics
	data.Captured, subDiags = types.MapValueFrom(ctx, types.StringType, map[string]string{})
	resp.Diagnostics.Append(subDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &data)
}

func (r *KubernetesWatch) retry(ctx context.Context, data *WatchModel, task func(context.Context, int) error) error {
	apiOptions, err := kresource.MergeAPIOptions(r.provider.DefaultApiOptions, data.ApiOptions.Options())
	if err != nil {
		return err
	}
	defaultHelper, err := apiOptions.Retry.NewHelper()
	if err != nil {
		return err
	}
	ctx, cancel := defaultHelper.SetDeadline(ctx)
	defer cancel()
	err = defaultHelper.Retry(ctx, task)
	if err != nil {
		return err
	}
	return nil
}
