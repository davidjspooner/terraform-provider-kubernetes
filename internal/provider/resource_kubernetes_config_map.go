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
var _ resource.Resource = &ConfigMap{}
var _ resource.ResourceWithImportState = &ConfigMap{}

func NewConfigMap() resource.Resource {
	return &ConfigMap{
		prometheusTypeNameSuffix: "_config_map",
	}
}

// ConfigMap defines the resource implementation.
type ConfigMap struct {
	provider                 *KubernetesProvider
	prometheusTypeNameSuffix string
}

// ConfigMapModel describes the resource data model.
type ConfigMapModel struct {
	MetaData   kresource.MetaData `tfsdk:"metadata"`
	Immutable  types.Bool         `tfsdk:"immutable"`
	Filenames  *FilesModel        `tfsdk:"file_data"`
	Data       types.Map          `tfsdk:"data"`
	BinaryData types.Map          `tfsdk:"binary_data"`
	Retry      *job.RetryModel    `tfsdk:"retry"`

	OutputMetadata
}

func (dmm *ConfigMapModel) Manifest() (unstructured.Unstructured, error) {

	sm := &StringMap{}
	sm.SetBase64Encoded(false)

	err := sm.AddFileModel(dmm.Filenames)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	err = sm.AddMaps(dmm.Data, dmm.BinaryData)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	var manifest unstructured.Unstructured
	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":        dmm.MetaData.Name,
			"labels":      dmm.MetaData.Labels,
			"annotations": dmm.MetaData.Annotations,
		},
		"data": sm.GetUnstructured(),
	})
	if dmm.MetaData.Namespace != nil {
		manifest.SetNamespace(*dmm.MetaData.Namespace)
	}
	immutable := dmm.Immutable.ValueBool()
	if immutable {
		manifest.Object["immutable"] = immutable
	}
	return manifest, nil
}

func (r *ConfigMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.prometheusTypeNameSuffix
}

func (r *ConfigMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Kubernetes ConfigMap resource. This resource manages the lifecycle of a Kubernetes configmap.",
		Attributes: map[string]schema.Attribute{
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the data cannot be updated",
				Optional:            true,
			},
			"retry":     job.RetryModelSchema(),
			"file_data": FileListSchema(false),
			"data": schema.MapAttribute{
				MarkdownDescription: "Data to store in the configmap",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"binary_data": schema.MapAttribute{
				MarkdownDescription: "Base64 encoded data to store in the configmap ( will be base64 decoded )",
				ElementType:         types.StringType,
				Optional:            true,
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
			"metadata": LongMetadataSchemaBlock(),
		},
	}
}

func (r *ConfigMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConfigMap) newCrudHelper(retryModel *job.RetryModel) (*CrudHelper, error) {
	helper := &CrudHelper{
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

func (r *ConfigMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConfigMapModel

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

	helper.Plan, err = plan.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Parsing manifest", err.Error())
		return
	}

	err = helper.CreateFromPlan(ctx,&plan.OutputMetadata)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource", err.Error())
		return
	}

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConfigMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConfigMapModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	helper, err := r.newCrudHelper(nil)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	helper.State, err = state.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource tfstate", err.Error())
		return
	}

	changed, err := helper.ReadActual(ctx, &state.OutputMetadata)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource", err.Error())
		return
	}

	if changed {
		// key := kresource.GetKey(stateResource)
		// actualUnstructured, err := r.provider.Shared.Get(ctx, key)
		//
		//	if err != nil {
		//		if kresource.ErrorIsNotFound(err) {
		//			//ok so we have to update the state to say it is stale
		//			resp.State.RemoveResource(ctx)
		//		} else {
		//			resp.Diagnostics.AddError("Failed to fetch resource", err.Error())
		//		}
		//	} else {
		//
		//		//compare state with current
		//		diffs, err := kresource.DiffResources(stateResource.Object, actualUnstructured.Object)
		//		for _, diff := range diffs {
		//			resp.Diagnostics.AddWarning(fmt.Sprintf("Read.Diff: %s", diff), "")
		//		}
		//		if err != nil {
		//			resp.Diagnostics.AddError("Failed to diff state and actual", err.Error())
		//			return
		//		} else {
		//			s, err := kresource.FormatYaml(actualUnstructured)
		//			if err != nil {
		//				resp.Diagnostics.AddError("Failed to format actual", err.Error())
		//				return
		//			}
		//			_ = s
		//		}
		//	}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *ConfigMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConfigMapModel
	var state ConfigMapModel

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

	helper.State, err = state.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource tfstate", err.Error())
		return
	}
	helper.Plan, err = plan.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource plan", err.Error())
		return
	}

	err = helper.Update(ctx,&plan.OutputMetadata)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConfigMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConfigMapModel

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

	helper.State, err = state.Manifest()
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

func (r *ConfigMap) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
