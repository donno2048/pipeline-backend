package service_test

//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/pipeline-backend/pkg/repository Repository
//go:generate mockgen -destination mock_model_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/model/v1alpha ModelServiceClient

import (
	"database/sql"
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

var ID = uuid.UUID{}
var Owner = "users/local-user"

func TestCreatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			ID:    "awesome",
			Owner: Owner,

			Recipe: &datamodel.Recipe{
				Source:      "connectors/http",
				Destination: "connectors/http",
			},

			Description: sql.NullString{
				String: "awesome pipeline",
				Valid:  true,
			},
		}

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetPipelineByID(gomock.Eq(normalPipeline.ID), gomock.Eq(normalPipeline.Owner)).
			Return(&normalPipeline, nil).
			Times(1)
		mockRepository.
			EXPECT().
			CreatePipeline(gomock.Eq(&normalPipeline)).
			Return(nil)

		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		s := service.NewService(mockRepository, mockModelServiceClient)

		_, err := s.CreatePipeline(&normalPipeline)

		assert.NoError(t, err)
	})
}

func TestUpdatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			ID:    "awesome",
			Owner: Owner,

			Description: sql.NullString{
				String: "awesome pipeline",
				Valid:  true,
			},
		}
		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetPipeline(gomock.Eq(ID), gomock.Eq(normalPipeline.Owner)).
			Return(&normalPipeline, nil).
			Times(1)
		mockRepository.
			EXPECT().
			GetPipelineByID(gomock.Eq(normalPipeline.ID), gomock.Eq(normalPipeline.Owner)).
			Return(&normalPipeline, nil).
			Times(1)
		mockRepository.
			EXPECT().
			UpdatePipeline(gomock.Eq(ID), gomock.Eq(Owner), gomock.Eq(&normalPipeline)).
			Return(nil)

		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		s := service.NewService(mockRepository, mockModelServiceClient)

		_, err := s.UpdatePipeline(ID, Owner, &normalPipeline)

		assert.NoError(t, err)
	})
}

func TestTriggerPipeline(t *testing.T) {
	t.Run("normal-url", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			ID:    "awesome",
			Owner: Owner,

			Description: sql.NullString{
				String: "awesome pipeline",
				Valid:  true,
			},

			Recipe: &datamodel.Recipe{
				Source:         "connectors/http",
				ModelInstances: []string{"models/yolov4/instances/latest"},
				Destination:    "connectors/http",
			},
		}

		mockRepository := NewMockRepository(ctrl)
		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		var pipelineInputs []*pipelinePB.Input
		pipelineInputs = append(pipelineInputs, &pipelinePB.Input{
			Type: &pipelinePB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/dog.jpg"},
		})

		s := service.NewService(mockRepository, mockModelServiceClient)

		_, err := s.TriggerPipeline(&pipelinePB.TriggerPipelineRequest{Inputs: pipelineInputs}, &normalPipeline)

		assert.NoError(t, err)
	})
}