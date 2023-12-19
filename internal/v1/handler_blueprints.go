package v1

import (
	"encoding/json"
	"net/http"

	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/osbuild/image-builder/internal/db"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type BlueprintV1 struct {
	Version        int            `json:"version"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Customizations Customizations `json:"customizations"`
	Distribution   Distributions  `json:"distribution"`
	ImageRequests  []ImageRequest `json:"image_requests"`
}

func BlueprintFromAPI(cbr CreateBlueprintRequest) BlueprintV1 {
	return BlueprintV1{
		Version:        1,
		Name:           cbr.Name,
		Description:    cbr.Description,
		Customizations: cbr.Customizations,
		Distribution:   cbr.Distribution,
		ImageRequests:  cbr.ImageRequests,
	}
}

func BlueprintFromEntry(be *db.BlueprintEntry) (BlueprintV1, error) {
	var result BlueprintV1
	err := json.Unmarshal(be.Body, &result)
	if err != nil {
		return BlueprintV1{}, err
	}
	result.Version = be.Version
	return result, nil
}

func (h *Handlers) CreateBlueprint(ctx echo.Context) error {
	idHeader, err := getIdentityHeader(ctx)
	if err != nil {
		return err
	}

	var blueprintRequest CreateBlueprintRequest
	err = ctx.Bind(&blueprintRequest)
	if err != nil {
		return err
	}

	blueprint := BlueprintFromAPI(blueprintRequest)
	body, err := json.Marshal(blueprint)
	if err != nil {
		return err
	}

	id := uuid.New()
	versionId := uuid.New()
	err = h.server.db.InsertBlueprint(id, versionId, idHeader.Identity.OrgID, idHeader.Identity.AccountNumber, blueprint.Name, blueprint.Description, body)
	if err != nil {
		logrus.Error("Error inserting id into db", err)
		return err
	}
	ctx.Logger().Infof("Inserted blueprint %s", id)
	return ctx.JSON(http.StatusCreated, ComposeResponse{
		Id: id,
	})
}

func (h *Handlers) UpdateBlueprint(ctx echo.Context, blueprintId uuid.UUID) error {
	idHeader, err := getIdentityHeader(ctx)
	if err != nil {
		return err
	}

	var blueprintRequest CreateBlueprintRequest
	err = ctx.Bind(&blueprintRequest)
	if err != nil {
		return err
	}

	blueprint := BlueprintFromAPI(blueprintRequest)
	body, err := json.Marshal(blueprint)
	if err != nil {
		return err
	}

	versionId := uuid.New()
	err = h.server.db.UpdateBlueprint(versionId, blueprintId, idHeader.Identity.OrgID, blueprint.Name, blueprint.Description, body)
	if err != nil {
		ctx.Logger().Errorf("Error updating blueprint in db: %v", err)
		return err
	}
	ctx.Logger().Infof("Updated blueprint %s", blueprintId)
	return ctx.JSON(http.StatusCreated, ComposeResponse{
		Id: blueprintId,
	})
}

func (h *Handlers) ComposeBlueprint(ctx echo.Context, id openapi_types.UUID) error {
	idHeader, err := getIdentityHeader(ctx)
	if err != nil {
		return err
	}

	blueprintEntry, err := h.server.db.GetBlueprint(id, idHeader.Identity.OrgID, idHeader.Identity.AccountNumber)
	if err != nil {
		return err
	}
	blueprint, err := BlueprintFromEntry(blueprintEntry)
	if err != nil {
		return err
	}
	composeResponses := make([]ComposeResponse, 0, len(blueprint.ImageRequests))
	clientId := ClientId("api")
	for _, imageRequest := range blueprint.ImageRequests {
		composeRequest := ComposeRequest{
			Customizations:   &blueprint.Customizations,
			Distribution:     blueprint.Distribution,
			ImageRequests:    []ImageRequest{imageRequest},
			ImageName:        &blueprint.Name,
			ImageDescription: &blueprint.Description,
			ClientId:         &clientId,
		}
		composesResponse, err := h.handleCommonCompose(ctx, composeRequest, &blueprintEntry.VersionId)
		if err != nil {
			return err
		}
		composeResponses = append(composeResponses, composesResponse)
	}

	return ctx.JSON(http.StatusCreated, composeResponses)
}
