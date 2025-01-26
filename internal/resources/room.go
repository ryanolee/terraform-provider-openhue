package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openhue/openhue-go"
	"github.com/ryanolee/terraform-provider-talk/internal/util"
)

type (
	Room struct {
		client *openhue.ClientWithResponses
	}

	RoomResourceModel struct {
		Name      types.String `tfsdk:"name"`
		Id        types.String `tfsdk:"id"`
		LightIds  types.List   `tfsdk:"lights"`
		Archetype types.String `tfsdk:"archetype"`
	}
)

func NewRoom() resource.Resource {
	return &Room{}
}

func (r *Room) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_room", req.ProviderTypeName)
}

func (r *Room) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *Room) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the room",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the room",
				Required:    true,
			},
			"archetype": schema.StringAttribute{
				Description: "The archetype of the room",
				Required:    true,
			},
			"lights": schema.ListAttribute{
				Description: "The lights in the room",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
		Description: "A room in the Hue system",
	}
}

func (r *Room) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to create room", "client is nil")
		return
	}

	var model RoomResourceModel

	resp.Diagnostics.Append(
		req.Plan.Get(ctx, &model)...,
	)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating room %s", model.Name.String()))
	putData, err := buildRoomPutPayload(model)
	if err != nil {
		resp.Diagnostics.AddError("failed to create room", fmt.Sprintf("failed to create room: %s", err.Error()))
		return
	}

	apiResp, err := r.client.CreateRoomWithResponse(ctx, putData)
	if err != nil {
		resp.Diagnostics.AddError("failed to create room", fmt.Sprintf("failed to create room: %s", err.Error()))
		return
	}

	if apiResp.HTTPResponse.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("failed to create room", fmt.Sprintf("failed to create room: %s, response data: %s", apiResp.HTTPResponse.Status, string(apiResp.Body)))
		return
	}

	if apiResp.Body == nil {
		resp.Diagnostics.AddError("failed to create room", "failed to create room: no response body")
		return
	}

	// Unmarshal the 201 response body onto the JSON200 struct
	// given it is a successful response
	json.Unmarshal(apiResp.Body, &apiResp.JSON200)
	responseData := *apiResp.JSON200.Data

	if len(responseData) == 0 {
		resp.Diagnostics.AddError("failed to create room", "failed to create room: no data in response body")
		return
	}

	model.Id = types.StringPointerValue(responseData[0].Rid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Room) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to read room", "client is nil")
		return
	}

	var model RoomResourceModel

	resp.Diagnostics.Append(
		req.State.Get(ctx, &model)...,
	)

	tflog.Info(ctx, fmt.Sprintf("Reading room %s", model.Id.String()))

	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.GetRoomWithResponse(ctx, model.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("failed to get room", fmt.Sprintf("failed to get room: %s", err.Error()))
		return
	}

	if apiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to get room", fmt.Sprintf("failed to get room: %s, %s", apiResp.HTTPResponse.Status, string(apiResp.Body)))
		return
	}

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("failed to get room", "failed to get room: no response body")
		return
	}
	responseData := *apiResp.JSON200.Data

	if len(responseData) == 0 {
		resp.Diagnostics.AddError("failed to get room", "failed to get room: no data in response body")
		return
	}

	model, err = mapRoomResponseToModel(ctx, responseData[0])
	if err != nil {
		resp.Diagnostics.AddError("failed to get room", fmt.Sprintf("failed to get room: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Room) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to update room", "client is nil")
		return
	}
	var model RoomResourceModel

	resp.Diagnostics.Append(
		req.Plan.Get(ctx, &model)...,
	)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Updating room %s", model.Id.ValueString()))

	putData, err := buildRoomPutPayload(model)
	apiResp, err := r.client.UpdateRoomWithResponse(ctx, model.Id.ValueString(), putData)
	if err != nil {
		resp.Diagnostics.AddError("failed to update room", fmt.Sprintf("failed to update room: %s", err.Error()))
		return
	}

	if apiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to update room", fmt.Sprintf("failed to update room: %s, %s", apiResp.HTTPResponse.Status, string(apiResp.Body)))
		return
	}

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("failed to update room", "failed to update room: no response body")
		return
	}
	responseData := *apiResp.JSON200.Data

	if len(responseData) == 0 {
		resp.Diagnostics.AddError("failed to update room", "failed to update room: no data in response body")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Room) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("failed to delete room", "client is nil")
		return
	}

	var model RoomResourceModel

	resp.Diagnostics.Append(
		req.State.Get(ctx, &model)...,
	)

	tflog.Info(ctx, fmt.Sprintf("Deleting room %s", model.Id.String()))

	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.DeleteRoomWithResponse(ctx, model.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to delete room", fmt.Sprintf("failed to delete room: %s", err.Error()))
		return
	}

	if apiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to delete room", fmt.Sprintf("failed to delete room: %s, %s", apiResp.HTTPResponse.Status, string(apiResp.Body)))
		return
	}

	return
}

var stringToRoomArchetype = map[string]openhue.RoomArchetype{
	"bedroom":  openhue.RoomArchetypeBedroom,
	"bathroom": openhue.RoomArchetypeBathroom,
	"kitchen":  openhue.RoomArchetypeKitchen,
	"dining":   openhue.RoomArchetypeDining,
	"office":   openhue.RoomArchetypeOffice,
	"other":    openhue.RoomArchetypeOther,
}

func mapStringToRoomArchetype(archetype string) (*openhue.RoomArchetype, bool) {
	val, ok := stringToRoomArchetype[archetype]
	return &val, ok
}

func buildRoomPutPayload(model RoomResourceModel) (openhue.RoomPut, error) {
	var putData openhue.RoomPut
	archetype, ok := mapStringToRoomArchetype(model.Archetype.ValueString())

	if !ok {
		return putData, fmt.Errorf("failed to create room: invalid archetype %s", model.Archetype.String())
	}

	// Build Metadata
	putData.Metadata = &struct {
		Archetype *openhue.RoomArchetype "json:\"archetype,omitempty\""
		Name      *string                "json:\"name,omitempty\""
	}{
		Name:      model.Name.ValueStringPointer(),
		Archetype: archetype,
	}

	// Build Children
	children := []openhue.ResourceIdentifier{}
	for _, lightId := range model.LightIds.Elements() {
		rid := lightId.(types.String).ValueString()
		children = append(children, openhue.ResourceIdentifier{
			Rid:   util.StringPointer(rid),
			Rtype: (*openhue.ResourceIdentifierRtype)(util.StringPointer("device")),
		})
	}
	putData.Children = &children

	return putData, nil
}

func mapRoomResponseToModel(ctx context.Context, room openhue.RoomGet) (RoomResourceModel, error) {
	var lightIds []string
	for _, light := range *room.Children {
		lightIds = append(lightIds, *light.Rid)
	}

	listValues, diagnostics := types.ListValueFrom(ctx, types.StringType, lightIds)
	if diagnostics.HasError() {
		return RoomResourceModel{}, fmt.Errorf("failed to map room response to model")
	}

	return RoomResourceModel{
		Name:      types.StringPointerValue(room.Metadata.Name),
		Id:        types.StringPointerValue(room.Id),
		LightIds:  listValues,
		Archetype: types.StringValue(string(*room.Metadata.Archetype)),
	}, nil
}
