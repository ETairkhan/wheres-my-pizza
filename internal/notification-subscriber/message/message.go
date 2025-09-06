package message

import (
	"encoding/json"

	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"
)

type MessageParser struct {
	logger *logger.Logger
}

func NewMessageParser(logger *logger.Logger) *MessageParser {
	return &MessageParser{
		logger: logger,
	}
}

func (p *MessageParser) ParseStatusUpdate(messageBytes []byte) (*models.StatusUpdateMessage, error) {
	var statusUpdate models.StatusUpdateMessage
	if err := json.Unmarshal(messageBytes, &statusUpdate); err != nil {
		return nil, err
	}
	return &statusUpdate, nil
}
