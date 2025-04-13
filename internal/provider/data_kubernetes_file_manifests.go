// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FileManifests{}

func NewFileManifests() datasource.DataSource {
	r := &FileManifests{
		prometheusTypeNameSuffix: "_manifest_files",
	}
	return r
}

// FileManifests defines the resource implementation.
type FileManifests struct {
	provider                 *KubernetesProvider
	prometheusTypeNameSuffix string
}

// FileManifestsModel describes the resource data model.
type FileManifestsModel struct {
	FileNames FilesModel `tfsdk:"file_data"`
	Manifests types.Map  `tfsdk:"manifests"`
}

func (r *FileManifests) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.prometheusTypeNameSuffix
}

func (r *FileManifests) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read yaml from a list of files and return all the inner manifests",

		Attributes: map[string]schema.Attribute{
			"file_data": FileListSchema(true),
			"manifests": ManifestMapSchema(),
		},
	}
}

func (r *FileManifests) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type manifestWithLineNumber struct {
	lineNumber int
	manifest   string
}

func (r *FileManifests) readManifestsFromReader(_ context.Context, reader io.Reader, _ *FileManifestsModel) []manifestWithLineNumber {
	scanner := bufio.NewScanner(reader)
	var manifest bytes.Buffer
	var manifests []manifestWithLineNumber
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		if line == "---" {
			if manifest.Len() >= 0 {
				manifests = append(manifests, manifestWithLineNumber{
					lineNumber: lineNumber,
					manifest:   manifest.String(),
				})
			}
			manifest.Reset()
			continue
		}
		manifest.WriteString(line)
		manifest.WriteString("\n")
	}
	if manifest.Len() >= 0 {
		manifests = append(manifests, manifestWithLineNumber{
			lineNumber: lineNumber,
			manifest:   manifest.String(),
		})
	}
	return manifests
}

func (r *FileManifests) expandTemplate(templateString string, values map[string]string) (string, error) {
	t := template.New("file")
	t, err := t.Parse(templateString)
	if err != nil {
		return "", err
	}
	var expanded bytes.Buffer
	err = t.Execute(&expanded, values)
	if err != nil {
		return "", err
	}
	return expanded.String(), nil
}

func (r *FileManifests) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileManifestsModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	sm := StringMap{}
	sm.AddFileModel(&config.FileNames)

	var values map[string]string
	var diags diag.Diagnostics
	diags = config.FileNames.Values.ElementsAs(ctx, &values, false)
	resp.Diagnostics.Append(diags...)

	allmanifests := make(map[string]attr.Value)

	if !resp.Diagnostics.HasError() {
		sm.ForEach(ctx, func(filename string, content string) error {
			var err error
			if len(values) > 0 {
				content, err = r.expandTemplate(content, values)
				if err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Failed to expand file %s", filename),
						err.Error(),
					)
					return err
				}
			}

			sr := strings.NewReader(content)
			manifests := r.readManifestsFromReader(ctx, sr, &config)
			for _, manifestWithLineNumber := range manifests {

				manifest, err := ReadManifest(manifestWithLineNumber.manifest)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						resp.Diagnostics.AddError(
							fmt.Sprintf("Failed to parse yaml from file %s line %d", filename, manifestWithLineNumber.lineNumber),
							err.Error(),
						)
					}
					continue
				}
				manifest.Source = fmt.Sprintf("%s:%d", filename, manifestWithLineNumber.lineNumber)
				key := manifest.Key()
				allmanifests[key], diags = types.ObjectValueFrom(ctx, ManifestType, manifest)
			}
			return nil
		})
	}

	// Copy allmanifests elements to config.Manifests
	config.Manifests, diags = basetypes.NewMapValue(types.ObjectType{AttrTypes: ManifestType}, allmanifests)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
