// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/davidjspooner/dsvalue/pkg/reflected"
	"github.com/davidjspooner/dsvalue/pkg/value"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesGet{}

func NewResourceGet() resource.Resource {
	r := &KubernetesGet{}
	return r
}

// KubernetesGet defines the resource implementation.
type KubernetesGet struct {
	provider *KubernetesProvider
}

// GetResourceModel describes the resource data model.
type GetResourceModel struct {
	ApiVersion types.String        `tfsdk:"api_version"`
	Kind       types.String        `tfsdk:"kind"`
	Metadata   *ShortMetadataModel `tfsdk:"metadata"`

	Querry QuerryList `tfsdk:"querry"`

	Retry *kresource.RetryModel `tfsdk:"retry"`

	Captured types.Map `tfsdk:"captured"`

	State       types.String `tfsdk:"state"`
	LastChecked types.String `tfsdk:"last_checked"`
}

func (r *KubernetesGet) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_get"
}

func (r *KubernetesGet) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"retry":  kresource.RetryModelSchema(),
			"querry": QuerrySchemaList(true),
			"captured": schema.MapAttribute{
				MarkdownDescription: "Captured data",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": ShortMetadataSchemaBlock(),
		},
	}
}

func (r *KubernetesGet) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.provider, ok = req.ProviderData.(*KubernetesProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubernetesProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
}

func (r *KubernetesGet) checvaluesOnce(ctx context.Context, key *kresource.Key, data *GetResourceModel) (*CaptureMap, error) {

	var captureMap CaptureMap
	var err error
	unstructured, err := r.provider.Shared.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	querryList := &data.Querry
	object, err := reflected.NewReflectedObject(reflect.ValueOf(unstructured.Object), value.UnknownSource)
	if err != nil {
		return nil, err
	}
	captureMap, err = querryList.Check(object)
	if err != nil {
		return nil, err
	}
	if len(captureMap) == 0 {
		return nil, fmt.Errorf("no querry match")
	}
	return &captureMap, nil
}

func (r *KubernetesGet) checvaluesWithRetry(ctx context.Context, key *kresource.Key, data *GetResourceModel, diags *diag.Diagnostics) {

	var savedCaptureMap *CaptureMap

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

func (r *KubernetesGet) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GetResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := kresource.Key{
		Kind:       data.Kind.ValueString(),
		ApiVersion: data.ApiVersion.ValueString(),
		MetaData: kresource.MetaData{
			Namespace: data.Metadata.Namespace.ValueString(),
			Name:      data.Metadata.Name.ValueString(),
		},
	}

	r.checvaluesWithRetry(ctx, &key, &data, &resp.Diagnostics)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesGet) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	key := kresource.Key{
		Kind:       data.Kind.ValueString(),
		ApiVersion: data.ApiVersion.ValueString(),
		MetaData: kresource.MetaData{
			Namespace: data.Metadata.Namespace.ValueString(),
			Name:      data.Metadata.Name.ValueString(),
		},
	}

	savedCaptureMap, err := r.checvaluesOnce(ctx, &key, &data)

	if err != nil {
		data.State = types.StringValue("stale")
	} else {
		var subDiags diag.Diagnostics
		data.Captured, subDiags = types.MapValueFrom(ctx, types.StringType, *savedCaptureMap)
		resp.Diagnostics.Append(subDiags...)
	}

	data.LastChecked = types.StringValue(time.Now().Format(time.RFC3339Nano))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesGet) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GetResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	key := kresource.Key{
		Kind:       data.Kind.ValueString(),
		ApiVersion: data.ApiVersion.ValueString(),
		MetaData: kresource.MetaData{
			Namespace: data.Metadata.Namespace.ValueString(),
			Name:      data.Metadata.Name.ValueString(),
		},
	}

	r.checvaluesWithRetry(ctx, &key, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesGet) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	//effectily a no-op - we just reset the output vars

	var data GetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var subDiags diag.Diagnostics
	data.Captured, subDiags = types.MapValueFrom(ctx, types.StringType, map[string]string{})
	resp.Diagnostics.Append(subDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &data)
}

func (r *KubernetesGet) retry(ctx context.Context, data *GetResourceModel, task func(context.Context, int) error) error {
	defaultHelper, err := data.Retry.NewHelper(nil)
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
