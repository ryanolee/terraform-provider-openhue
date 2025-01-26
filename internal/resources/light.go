package resources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openhue/openhue-go"
)

type (
	Light struct {
		client *openhue.ClientWithResponses
	}

	lightResourceModel struct {
		Name       types.String            `tfsdk:"name"`
		Id         types.String            `tfsdk:"id"`
		On         types.Bool              `tfsdk:"on"`
		Brightness types.Float32           `tfsdk:"brightness"`
		Color      lightResourceModelColor `tfsdk:"color"`
	}

	lightResourceModelColor struct {
		X types.Float32 `tfsdk:"x"`
		Y types.Float32 `tfsdk:"y"`
		Z types.Float32 `tfsdk:"z"`
	}
)

func NewLight() resource.Resource {
	return &Light{}
}

func (r *Light) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_light", req.ProviderTypeName)
}

func (r *Light) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure can be called multiple times (sometimes without provider data)
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*openhue.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("expected openhue.ClientWithResponses", fmt.Sprintf("Expected *openhue.Client, got %T", req.ProviderData))
		return
	}

	r.client = client
}

func (r *Light) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the light",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the light. This MUST match the name of the light in the Hue system exactly in order for the provider to find the light.",
				Required:    true,
			},
			"on": schema.BoolAttribute{
				Description: "Whether the light is on or off",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"brightness": schema.Float32Attribute{
				Description: "The brightness of the light",
				Optional:    true,
			},
			"color": schema.ObjectAttribute{
				Description: "The color of the light",
				Optional:    true,
				Computed:    true,
				AttributeTypes: map[string]attr.Type{
					"x": types.Float32Type,
					"y": types.Float32Type,
					"z": types.Float32Type,
				},
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"x": types.Float32Type,
							"y": types.Float32Type,
							"z": types.Float32Type,
						},
						map[string]attr.Value{
							"x": types.Float32Value(0.0),
							"y": types.Float32Value(0.0),
							"z": types.Float32Value(0.0),
						},
					),
				),
			},
		},
		Description: "A light in the Hue system",
	}
}

func (r *Light) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to create light", "client is nil")
		return
	}

	var model lightResourceModel

	resp.Diagnostics.Append(
		req.Plan.Get(ctx, &model)...,
	)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedApiResp, err := r.client.GetLightsWithResponse(ctx)

	if err != nil {
		resp.Diagnostics.AddError("failed to find light", fmt.Sprintf("failed to create light: %s", err.Error()))
		return
	}

	if parsedApiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to find light", fmt.Sprintf("failed to create light: %s, %s", parsedApiResp.HTTPResponse.Status, string(parsedApiResp.Body)))
		return
	}

	getLightResponse := *parsedApiResp.JSON200.Data

	// Search for the light in the response
	var targetLight *openhue.LightGet
	for _, light := range getLightResponse {
		if *light.Metadata.Name == model.Name.ValueString() {
			targetLight = &light
			break
		}
	}

	if targetLight == nil {
		lampNames := make([]string, len(getLightResponse))
		for i, light := range getLightResponse {
			lampNames[i] = *light.Metadata.Name
		}

		resp.Diagnostics.AddError("failed to find light", fmt.Sprintf("failed to create light: light %s not found. Available lamps %s", model.Name.String(), strings.Join(lampNames, ", ")))
		return
	}

	// Set the ID of the light
	model.Id = types.StringPointerValue(targetLight.Id)

	// Update light to reflect new state
	tflog.Info(ctx, fmt.Sprintf("Updating light %s", model.Id.String()))

	lightPut, err := lightModelToPayload(&model)
	if lightPut == nil {
		resp.Diagnostics.AddError("failed to update light", "failed to update light: failed to parse color")
		return
	}

	updateResp, err := r.client.UpdateLightWithResponse(ctx, model.Id.ValueString(), *lightPut)
	if err != nil {
		resp.Diagnostics.AddError("failed to update light", fmt.Sprintf("failed to update light: %s", err.Error()))
		return
	}

	if updateResp.HTTPResponse.StatusCode != http.StatusOK && updateResp.HTTPResponse.StatusCode != http.StatusMultiStatus {
		resp.Diagnostics.AddError("failed to update light", fmt.Sprintf("failed to update light: %s, %s", updateResp.HTTPResponse.Status, string(updateResp.Body)))
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("failed to update light", fmt.Sprintf("failed to update light: no parsed response body.", string(updateResp.Body)))
		return
	}

	responseData := *updateResp.JSON200.Data

	if len(responseData) == 0 {
		resp.Diagnostics.AddError("failed to get light", "failed to get light: no data in response body")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Light) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to read light", "client is nil")
		return
	}

	var model lightResourceModel

	resp.Diagnostics.Append(
		req.State.Get(ctx, &model)...,
	)

	tflog.Info(ctx, fmt.Sprintf("Reading light %s", model.Id.String()))

	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.GetLightWithResponse(ctx, model.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to get light", fmt.Sprintf("failed to get light: %s", err.Error()))
		return
	}

	if apiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to get light", fmt.Sprintf("failed to get light: %s, %s", apiResp.HTTPResponse.Status, string(apiResp.Body)))
		return
	}

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("failed to get light", "failed to get light: no response body")
		return
	}
	responseData := *apiResp.JSON200.Data

	if len(responseData) == 0 {
		resp.Diagnostics.AddError("failed to get light", "failed to get light: no data in response body")
		return
	}

	model = mapLightStateToModel(model, &responseData[0])

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Light) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to update light", "client is nil")
		return
	}

	var model lightResourceModel

	resp.Diagnostics.Append(
		req.Plan.Get(ctx, &model)...,
	)

	if resp.Diagnostics.HasError() {
		return
	}

	lightPut, err := lightModelToPayload(&model)
	if err != nil {
		resp.Diagnostics.AddError("failed to update light", fmt.Sprintf("failed to update light: %s", err.Error()))
		return
	}

	apiResp, err := r.client.UpdateLightWithResponse(ctx, model.Id.ValueString(), *lightPut)
	if err != nil {
		resp.Diagnostics.AddError("failed to update light", fmt.Sprintf("failed to update light: %s", err.Error()))
		return
	}

	if apiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to update light", fmt.Sprintf("failed to update light: %s, %s", apiResp.HTTPResponse.Status, string(apiResp.Body)))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Light) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// This is a no-op because lights cannot be deleted
	return
}

func lightModelToPayload(model *lightResourceModel) (*openhue.LightPut, error) {
	return &openhue.LightPut{
		On: &openhue.On{
			On: model.On.ValueBoolPointer(),
		},
		Color: &openhue.Color{
			Xy: &openhue.GamutPosition{
				X: model.Color.X.ValueFloat32Pointer(),
				Y: model.Color.Y.ValueFloat32Pointer(),
			},
		},
		Dimming: &openhue.Dimming{
			Brightness: model.Brightness.ValueFloat32Pointer(),
		},
	}, nil
}

func mapLightStateToModel(lightModel lightResourceModel, light *openhue.LightGet) lightResourceModel {
	return lightResourceModel{
		Name: types.StringPointerValue(light.Metadata.Name),
		Id:   types.StringPointerValue(light.Id),
		On:   types.BoolPointerValue(light.On.On),
		Brightness: types.Float32PointerValue(
			light.Dimming.Brightness,
		),
		Color: lightModel.Color,
	}
}
