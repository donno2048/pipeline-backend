package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func IsConnector(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connector-resources/")
}
func IsConnectorWithNamespace(resourceName string) bool {
	return len(strings.Split(resourceName, "/")) > 3 && strings.Split(resourceName, "/")[2] == "connector-resources"
}

func IsConnectorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connector-definitions/")
}

func IsOperatorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "operator-definitions/")
}

func (s *service) recipeNameToPermalink(userUid uuid.UUID, recipeRscName *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipePermalink := &pipelinePB.Recipe{Version: recipeRscName.Version}
	for _, component := range recipeRscName.Components {
		componentPermalink := &pipelinePB.Component{
			Id:            component.Id,
			Configuration: component.Configuration,
		}

		permalink := ""
		var err error
		if IsConnectorWithNamespace(component.ResourceName) {
			permalink, err = s.connectorNameToPermalink(userUid, component.ResourceName)
			if err != nil {
				// Allow not created resource
				componentPermalink.ResourceName = ""
			} else {
				componentPermalink.ResourceName = permalink
			}
		}

		if IsConnectorDefinition(component.DefinitionName) {
			permalink, err = s.connectorDefinitionNameToPermalink(component.DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.DefinitionName = permalink
		} else if IsOperatorDefinition(component.DefinitionName) {
			permalink, err = s.operatorDefinitionNameToPermalink(component.DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.DefinitionName = permalink
		}

		recipePermalink.Components = append(recipePermalink.Components, componentPermalink)
	}
	return recipePermalink, nil
}

func (s *service) recipePermalinkToName(userUid uuid.UUID, recipePermalink *datamodel.Recipe) (*datamodel.Recipe, error) {

	recipe := &datamodel.Recipe{Version: recipePermalink.Version}

	for _, componentPermalink := range recipePermalink.Components {
		component := &datamodel.Component{
			Id:            componentPermalink.Id,
			Configuration: componentPermalink.Configuration,
		}

		if IsConnector(componentPermalink.ResourceName) {
			name, err := s.connectorPermalinkToName(userUid, componentPermalink.ResourceName)
			if err != nil {
				// Allow resource not created
				component.ResourceName = ""
			} else {
				component.ResourceName = name
			}
		}
		if IsConnectorDefinition(componentPermalink.DefinitionName) {
			name, err := s.connectorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		} else if IsOperatorDefinition(componentPermalink.DefinitionName) {
			name, err := s.operatorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		}

		recipe.Components = append(recipe.Components, component)
	}
	return recipe, nil
}

func (s *service) recipePermalinkToNameAdmin(recipePermalink *datamodel.Recipe) (*datamodel.Recipe, error) {

	recipe := &datamodel.Recipe{Version: recipePermalink.Version}

	for _, componentPermalink := range recipePermalink.Components {
		component := &datamodel.Component{
			Id:            componentPermalink.Id,
			Configuration: componentPermalink.Configuration,
		}

		if IsConnector(componentPermalink.ResourceName) {
			name, err := s.connectorPermalinkToNameAdmin(componentPermalink.ResourceName)
			if err != nil {
				// Allow resource not created
				component.ResourceName = ""
			} else {
				component.ResourceName = name
			}
		}
		if IsConnectorDefinition(componentPermalink.DefinitionName) {
			name, err := s.connectorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		} else if IsOperatorDefinition(componentPermalink.DefinitionName) {
			name, err := s.operatorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		}

		recipe.Components = append(recipe.Components, component)
	}
	return recipe, nil
}

func (s *service) connectorNameToPermalink(userUid uuid.UUID, name string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = utils.InjectOwnerToContextWithUserUid(ctx, userUid)

	resp, err := s.connectorPublicServiceClient.GetUserConnectorResource(ctx,
		&connectorPB.GetUserConnectorResourceRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", name, err)
	}

	return fmt.Sprintf("connector-resources/%s", resp.ConnectorResource.Uid), nil
}

func (s *service) connectorPermalinkToName(userUid uuid.UUID, permalink string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = utils.InjectOwnerToContextWithUserUid(ctx, userUid)

	resp, err := s.connectorPublicServiceClient.LookUpConnectorResource(ctx,
		&connectorPB.LookUpConnectorResourceRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpConnector1", permalink, err)
	}

	return resp.ConnectorResource.Name, nil
}

func (s *service) connectorPermalinkToNameAdmin(permalink string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.connectorPrivateServiceClient.LookUpConnectorResourceAdmin(ctx,
		&connectorPB.LookUpConnectorResourceAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpConnector2", permalink, err)
	}

	return resp.ConnectorResource.Name, nil
}

func (s *service) connectorDefinitionNameToPermalink(name string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.connectorPublicServiceClient.GetConnectorDefinition(ctx,
		&connectorPB.GetConnectorDefinitionRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnectorDefinition", name, err)
	}

	return fmt.Sprintf("connector-definitions/%s", resp.ConnectorDefinition.Uid), nil
}

func (s *service) connectorDefinitionPermalinkToName(permalink string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.connectorPrivateServiceClient.LookUpConnectorDefinitionAdmin(ctx,
		&connectorPB.LookUpConnectorDefinitionAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpConnectorDefinitionAdmin", permalink, err)
	}

	return resp.ConnectorDefinition.Name, nil
}

func (s *service) operatorDefinitionNameToPermalink(name string) (string, error) {
	id, err := resource.GetRscNameID(name)
	if err != nil {
		return "", err
	}
	def, err := s.operator.GetOperatorDefinitionById(id)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("operator-definitions/%s", def.Uid), nil
}

func (s *service) operatorDefinitionPermalinkToName(permalink string) (string, error) {
	uid, err := resource.GetRscPermalinkUID(permalink)
	if err != nil {
		return "", err
	}
	def, err := s.operator.GetOperatorDefinitionByUid(uid)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpOperatorDefinitionAdmin", permalink, err)
	}

	return fmt.Sprintf("operator-definitions/%s", def.Id), nil
}

func ConvertResourceUIDToControllerResourcePermalink(resourceUID uuid.UUID, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID.String(), resourceType)

	return resourcePermalink
}

func (s *service) includeDetailInRecipe(recipe *pipelinePB.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		if IsConnectorWithNamespace(recipe.Components[idx].ResourceName) {
			resp, err := s.connectorPublicServiceClient.GetUserConnectorResource(ctx, &connectorPB.GetUserConnectorResourceRequest{
				Name: recipe.Components[idx].ResourceName,
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				// Allow resource not created
				recipe.Components[idx].Resource = nil
			} else {
				detail := &structpb.Struct{}
				// Note: need to deal with camelCase or under_score for grpc in future
				json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnectorResource())
				if marshalErr != nil {
					return marshalErr
				}
				unmarshalErr := detail.UnmarshalJSON(json)
				if unmarshalErr != nil {
					return unmarshalErr
				}

				recipe.Components[idx].Resource = resp.ConnectorResource
			}

		}
		if IsConnectorDefinition(recipe.Components[idx].DefinitionName) {
			resp, err := s.connectorPublicServiceClient.GetConnectorDefinition(ctx, &connectorPB.GetConnectorDefinitionRequest{
				Name: recipe.Components[idx].DefinitionName,
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnectorDefinition", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnectorDefinition())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_ConnectorDefinition{ConnectorDefinition: resp.ConnectorDefinition}
		}
		if IsOperatorDefinition(recipe.Components[idx].DefinitionName) {
			id, err := resource.GetRscNameID(recipe.Components[idx].DefinitionName)
			if err != nil {
				return err
			}
			def, err := s.operator.GetOperatorDefinitionById(id)
			if err != nil {
				return err
			}

			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(def)
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_OperatorDefinition{OperatorDefinition: def}
		}

	}
	return nil
}

// PBToDBPipeline converts protobuf data model to db data model
func (s *service) PBToDBPipeline(ctx context.Context, userUid uuid.UUID, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var owner string
	var err error

	switch pbPipeline.Owner.(type) {
	case *pipelinePB.Pipeline_User:
		owner, err = s.ConvertOwnerNameToPermalink(pbPipeline.GetUser())
		if err != nil {
			return nil, err
		}
	case *pipelinePB.Pipeline_Org:

		return nil, fmt.Errorf("org not supported")
	}

	recipe := &datamodel.Recipe{}
	if pbPipeline.GetRecipe() != nil {
		recipePermalink, err := s.recipeNameToPermalink(userUid, pbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(recipePermalink)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &recipe); err != nil {
			return nil, err
		}

	}

	return &datamodel.Pipeline{
		Owner: owner,
		ID:    pbPipeline.GetId(),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipeline.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipeline.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipeline.GetCreateTime() != nil {
					return pbPipeline.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipeline.GetUpdateTime() != nil {
					return pbPipeline.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipeline.GetDescription(),
			Valid:  true,
		},

		Recipe:     recipe,
		Visibility: datamodel.PipelineVisibility(pbPipeline.Visibility),
	}, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipeline(ctx context.Context, userUid uuid.UUID, dbPipeline *datamodel.Pipeline, view pipelinePB.View) (*pipelinePB.Pipeline, error) {

	owner, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}

	var pbRecipe *pipelinePB.Recipe
	if dbPipeline.Recipe != nil {
		pbRecipe = &pipelinePB.Recipe{}
		recipeRscName, err := s.recipePermalinkToName(userUid, dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := json.Marshal(recipeRscName)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}

		for i := range pbRecipe.Components {
			// TODO: use enum
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions/") {
				if pbRecipe.Components[i].Resource != nil {
					switch pbRecipe.Components[i].Resource.Type {
					case connectorPB.ConnectorType_CONNECTOR_TYPE_AI:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
					case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
					case connectorPB.ConnectorType_CONNECTOR_TYPE_DATA:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
					}
				}
			} else if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions/") {
				pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
			}
		}
	}

	pbPipeline := pipelinePB.Pipeline{
		Name:        fmt.Sprintf("%s/pipelines/%s", owner, dbPipeline.ID),
		Uid:         dbPipeline.BaseDynamic.UID.String(),
		Id:          dbPipeline.ID,
		CreateTime:  timestamppb.New(dbPipeline.CreateTime),
		UpdateTime:  timestamppb.New(dbPipeline.UpdateTime),
		Description: &dbPipeline.Description.String,
		Visibility:  pipelinePB.Visibility(dbPipeline.Visibility),
		Recipe:      pbRecipe,
	}

	if strings.HasPrefix(dbPipeline.Owner, "users/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_User{User: owner}
	} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_Org{Org: owner}
	}
	if view == pipelinePB.View_VIEW_FULL {
		if err := s.includeDetailInRecipe(pbPipeline.Recipe); err != nil {
			return nil, err
		}
	}

	return &pbPipeline, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipelines(ctx context.Context, userUid uuid.UUID, dbPipelines []*datamodel.Pipeline, view pipelinePB.View) ([]*pipelinePB.Pipeline, error) {
	var err error
	pbPipelines := make([]*pipelinePB.Pipeline, len(dbPipelines))
	for idx := range dbPipelines {
		pbPipelines[idx], err = s.DBToPBPipeline(
			ctx,
			userUid,
			dbPipelines[idx],
			view,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelines, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipelineAdmin(ctx context.Context, dbPipeline *datamodel.Pipeline, view pipelinePB.View) (*pipelinePB.Pipeline, error) {

	owner, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	var pbRecipe *pipelinePB.Recipe
	if dbPipeline.Recipe != nil {
		pbRecipe = &pipelinePB.Recipe{}
		recipeRscName, err := s.recipePermalinkToNameAdmin(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := json.Marshal(recipeRscName)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}

		for i := range pbRecipe.Components {
			// TODO: use enum
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions/") {
				if pbRecipe.Components[i].Resource != nil {
					switch pbRecipe.Components[i].Resource.Type {
					case connectorPB.ConnectorType_CONNECTOR_TYPE_AI:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
					case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
					case connectorPB.ConnectorType_CONNECTOR_TYPE_DATA:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
					}
				}
			} else if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions/") {
				pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
			}
		}

	}

	pbPipeline := pipelinePB.Pipeline{
		Name:        fmt.Sprintf("%s/pipelines/%s", owner, dbPipeline.ID),
		Uid:         dbPipeline.BaseDynamic.UID.String(),
		Id:          dbPipeline.ID,
		CreateTime:  timestamppb.New(dbPipeline.CreateTime),
		UpdateTime:  timestamppb.New(dbPipeline.UpdateTime),
		Description: &dbPipeline.Description.String,
		Visibility:  pipelinePB.Visibility(dbPipeline.Visibility),
		Recipe:      pbRecipe,
	}

	if strings.HasPrefix(dbPipeline.Owner, "users/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_User{User: owner}
	} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_Org{Org: owner}
	}
	if view == pipelinePB.View_VIEW_FULL {
		if err := s.includeDetailInRecipe(pbPipeline.Recipe); err != nil {
			return nil, err
		}
	}

	return &pbPipeline, nil
}

// DBToPBPipeline converts db data model to protobuf data model
// TODO: refactor this
func (s *service) DBToPBPipelinesAdmin(ctx context.Context, dbPipelines []*datamodel.Pipeline, view pipelinePB.View) ([]*pipelinePB.Pipeline, error) {
	var err error
	pbPipelines := make([]*pipelinePB.Pipeline, len(dbPipelines))
	for idx := range dbPipelines {
		pbPipelines[idx], err = s.DBToPBPipelineAdmin(
			ctx,
			dbPipelines[idx],
			view,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelines, nil
}

// PBToDBPipelineRelease converts protobuf data model to db data model
func (s *service) PBToDBPipelineRelease(ctx context.Context, userUid uuid.UUID, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipelineRelease.GetRecipe() != nil {
		recipePermalink, err := s.recipeNameToPermalink(userUid, pbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(recipePermalink)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &recipe); err != nil {
			return nil, err
		}

	}
	return &datamodel.PipelineRelease{
		ID: pbPipelineRelease.GetId(),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipelineRelease.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipelineRelease.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipelineRelease.GetCreateTime() != nil {
					return pbPipelineRelease.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipelineRelease.GetUpdateTime() != nil {
					return pbPipelineRelease.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipelineRelease.GetDescription(),
			Valid:  true,
		},

		Recipe:      recipe,
		PipelineUID: pipelineUid,
		Visibility:  datamodel.PipelineVisibility(pbPipelineRelease.Visibility),
	}, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
func (s *service) DBToPBPipelineRelease(ctx context.Context, userUid uuid.UUID, dbPipelineRelease *datamodel.PipelineRelease, view pipelinePB.View) (*pipelinePB.PipelineRelease, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, dbPipelineRelease.PipelineUID, true)
	if err != nil {
		return nil, err
	}
	owner, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	var pbRecipe *pipelinePB.Recipe
	if dbPipelineRelease.Recipe != nil {
		pbRecipe = &pipelinePB.Recipe{}
		recipeRscName, err := s.recipePermalinkToName(userUid, dbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(recipeRscName)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}

		for i := range pbRecipe.Components {
			// TODO: use enum
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions/") {
				if pbRecipe.Components[i].Resource != nil {
					switch pbRecipe.Components[i].Resource.Type {
					case connectorPB.ConnectorType_CONNECTOR_TYPE_AI:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
					case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
					case connectorPB.ConnectorType_CONNECTOR_TYPE_DATA:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
					}
				}
			} else if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions/") {
				pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
			}
		}
	}

	pbPipelineRelease := pipelinePB.PipelineRelease{
		Name:        fmt.Sprintf("%s/pipelines/%s/releases/%s", owner, dbPipeline.ID, dbPipelineRelease.ID),
		Uid:         dbPipelineRelease.BaseDynamic.UID.String(),
		Id:          dbPipelineRelease.ID,
		CreateTime:  timestamppb.New(dbPipelineRelease.CreateTime),
		UpdateTime:  timestamppb.New(dbPipelineRelease.UpdateTime),
		Description: &dbPipelineRelease.Description.String,
		Visibility:  pipelinePB.Visibility(dbPipeline.Visibility),
		Recipe:      pbRecipe,
	}

	if view == pipelinePB.View_VIEW_FULL {
		if err := s.includeDetailInRecipe(pbPipelineRelease.Recipe); err != nil {
			return nil, err
		}
	}

	return &pbPipelineRelease, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
func (s *service) DBToPBPipelineReleases(ctx context.Context, userUid uuid.UUID, dbPipelineRelease []*datamodel.PipelineRelease, view pipelinePB.View) ([]*pipelinePB.PipelineRelease, error) {
	var err error
	pbPipelineReleases := make([]*pipelinePB.PipelineRelease, len(dbPipelineRelease))
	for idx := range dbPipelineRelease {
		pbPipelineReleases[idx], err = s.DBToPBPipelineRelease(
			ctx,
			userUid,
			dbPipelineRelease[idx],
			view,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelineReleases, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
// TODO: refactor this
func (s *service) DBToPBPipelineReleaseAdmin(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view pipelinePB.View) (*pipelinePB.PipelineRelease, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, dbPipelineRelease.PipelineUID, true)
	if err != nil {
		return nil, err
	}
	owner, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	var pbRecipe *pipelinePB.Recipe
	if dbPipelineRelease.Recipe != nil {
		pbRecipe = &pipelinePB.Recipe{}
		recipeRscName, err := s.recipePermalinkToNameAdmin(dbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(recipeRscName)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}

		for i := range pbRecipe.Components {
			// TODO: use enum
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions/") {
				if pbRecipe.Components[i].Resource != nil {
					switch pbRecipe.Components[i].Resource.Type {
					case connectorPB.ConnectorType_CONNECTOR_TYPE_AI:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
					case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
					case connectorPB.ConnectorType_CONNECTOR_TYPE_DATA:
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
					}
				}
			} else if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions/") {
				pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
			}
		}

	}
	pbPipelineRelease := pipelinePB.PipelineRelease{
		Name:        fmt.Sprintf("%s/pipelines/%s/releases/%s", owner, dbPipeline.ID, dbPipelineRelease.ID),
		Uid:         dbPipelineRelease.BaseDynamic.UID.String(),
		Id:          dbPipelineRelease.ID,
		CreateTime:  timestamppb.New(dbPipelineRelease.CreateTime),
		UpdateTime:  timestamppb.New(dbPipelineRelease.UpdateTime),
		Description: &dbPipelineRelease.Description.String,
		Visibility:  pipelinePB.Visibility(dbPipeline.Visibility),
		Recipe:      pbRecipe,
	}

	if view == pipelinePB.View_VIEW_FULL {
		if err := s.includeDetailInRecipe(pbPipelineRelease.Recipe); err != nil {
			return nil, err
		}
	}

	return &pbPipelineRelease, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
// TODO: refactor this
func (s *service) DBToPBPipelineReleasesAdmin(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view pipelinePB.View) ([]*pipelinePB.PipelineRelease, error) {
	var err error
	pbPipelineReleases := make([]*pipelinePB.PipelineRelease, len(dbPipelineRelease))
	for idx := range dbPipelineRelease {
		pbPipelineReleases[idx], err = s.DBToPBPipelineReleaseAdmin(
			ctx,
			dbPipelineRelease[idx],
			view,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelineReleases, nil
}
