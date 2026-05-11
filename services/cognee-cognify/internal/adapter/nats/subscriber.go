package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type DataIngestedSubscriber struct {
	nc             *nats.Conn
	cognifyUseCase port.CognifyUseCase
	logger         *slog.Logger
}

func NewDataIngestedSubscriber(nc *nats.Conn, uc port.CognifyUseCase, logger *slog.Logger) *DataIngestedSubscriber {
	return &DataIngestedSubscriber{
		nc:             nc,
		cognifyUseCase: uc,
		logger:         logger,
	}
}

func (s *DataIngestedSubscriber) Start() error {
	_, err := s.nc.Subscribe("cognee.data.ingested", func(msg *nats.Msg) {
		ctx := context.Background()
		var event domain.DataIngestedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("Failed to unmarshal data.ingested event", "error", err)
			return
		}

		datasetID, err := uuid.Parse(event.DatasetID)
		if err != nil {
			s.logger.Error("Invalid dataset ID", "error", err)
			return
		}

		s.logger.Info("Received data.ingested event, triggering pipeline", "dataset_id", datasetID)

		req := dto.TriggerCognifyReq{
			DatasetID: datasetID,
			TenantID:  event.TenantID,
			Config:    domain.DefaultCognifyConfig(),
		}

		if _, err := s.cognifyUseCase.Execute(ctx, req); err != nil {
			s.logger.Error("Pipeline execution failed", "dataset_id", datasetID, "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to cognee.data.ingested: %w", err)
	}
	s.logger.Info("Subscribed to cognee.data.ingested")
	return nil
}
