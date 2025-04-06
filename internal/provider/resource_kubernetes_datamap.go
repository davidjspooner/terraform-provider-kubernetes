// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DataMap{}
var _ resource.ResourceWithImportState = &DataMap{}

func NewConfigMap() resource.Resource {
	return &DataMap{secret: false}
}

func NewSecret() resource.Resource {
	return &DataMap{secret: true}
}

// DataMap defines the resource implementation.
type DataMap struct {
	provider *KubernetesProvider
	secret   bool
}

// DataMapModel describes the resource data model.
type DataMapModel struct {
	MetaData  kresource.MetaData `tfsdk:"metadata"`
	Type      types.String       `tfsdk:"type"`
	Immutable types.Bool         `tfsdk:"immutable"`
	Data      types.Map          `tfsdk:"data"`
	Retry     *job.RetryModel    `tfsdk:"retry"`
}

func (dmm *DataMapModel) Manifest(secret bool) (unstructured.Unstructured, error) {
	var manifest unstructured.Unstructured
	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":        dmm.MetaData.Name,
			"namespace":   dmm.MetaData.Namespace,
			"labels":      dmm.MetaData.Labels,
			"annotations": dmm.MetaData.Annotations,
		},
		"data": dmm.Data,
	})
	if secret {
		manifest.SetKind("Secret")
	}
	return manifest, nil
}

func (r *DataMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	if r.secret {
		resp.TypeName = req.ProviderTypeName + "_secret"
	} else {
		resp.TypeName = req.ProviderTypeName + "_config_map"
	}
}

func (r *DataMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the data cannot be updated",
				Optional:            true,
			},
			"retry": job.RetryModelSchema(),
		},
		Blocks: map[string]schema.Block{
			"metadata": LongMetadataSchemaBlock(),
		},
	}
	if r.secret {
		resp.Schema.MarkdownDescription = "Kubernetes Secret"
		resp.Schema.Attributes["data"] = schema.MapAttribute{
			MarkdownDescription: "Data to store in the secret",
			ElementType:         types.StringType,
			Optional:            false,
		}
		resp.Schema.Attributes["type"] = schema.StringAttribute{
			MarkdownDescription: "Type of the secret",
			Optional:            true,
		}
	} else {
		resp.Schema.MarkdownDescription = "Kubernetes ConfigMap"
		resp.Schema.Attributes["data"] = schema.MapAttribute{
			MarkdownDescription: "Data to store in the configmap",
			ElementType:         types.StringType,
			Optional:            false,
		}
	}
}

func (r *DataMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DataMap) newCrudHelper(retryModel *job.RetryModel) (*kresource.CrudHelper, error) {
	helper := &kresource.CrudHelper{
		Shared: &r.provider.Shared,
	}
	var err error
	if retryModel != nil {
		helper.RetryHelper, err = retryModel.NewHelper(r.provider.DefaultRetry)
		if err != nil {
			return nil, err
		}
	} else if r.provider.DefaultRetry != nil {
		helper.RetryHelper = r.provider.DefaultRetry
	} else {
		helper.RetryHelper = &job.RetryHelper{}
	}
	return helper, nil
}

func (r *DataMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DataMapModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helper, err := r.newCrudHelper(plan.Retry)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	helper.Plan, err = plan.Manifest(r.secret)
	if err != nil {
		resp.Diagnostics.AddError("Parsing manifest", err.Error())
		return
	}

	err = helper.CreateFromPlan(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource", err.Error())
		return
	}

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DataMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DataMapModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	stateResource, err := state.Manifest(r.secret)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource tfstate", err.Error())
		return
	}
	key := kresource.GetKey(stateResource)
	actualUnstructured, err := r.provider.Shared.Get(ctx, key)

	if err != nil {
		if kresource.ErrorIsNotFound(err) {
			//ok so we have to update the state to say it is stale
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Failed to fetch resource", err.Error())
		}
	} else {
		//compare state with current
		diffs, err := kresource.DiffResources(stateResource.Object, actualUnstructured.Object)
		for _, diff := range diffs {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Read.Diff: %s", diff), "")
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to diff state and actual", err.Error())
			return
		} else {
			_, err := kresource.FormatYaml(actualUnstructured)
			if err != nil {
				resp.Diagnostics.AddError("Failed to format actual", err.Error())
				return
			}
		}
	}
}

func (r *DataMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DataMapModel
	var state DataMapModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helper, err := r.newCrudHelper(plan.Retry)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	helper.State, err = state.Manifest(r.secret)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource tfstate", err.Error())
		return
	}
	helper.Plan, err = plan.Manifest(r.secret)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource plan", err.Error())
		return
	}

	err = helper.Update(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DataMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DataMapModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helper, err := r.newCrudHelper(state.Retry)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	helper.State, err = state.Manifest(r.secret)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse reource", err.Error())
		return
	}

	err = helper.DeleteState(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Destroying resource", err.Error())
		return
	}
	req.State.RemoveResource(ctx)
}

func (r *DataMap) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
