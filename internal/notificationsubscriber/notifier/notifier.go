package notifier

import (
	"fmt"
	"time"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"
)

type Notifier struct {
	logger *logger.Logger
}

func NewNotifier(logger *logger.Logger) *Notifier {
	return &Notifier{
		logger: logger,
	}
}

func (n *Notifier) DisplayNotification(statusUpdate *models.StatusUpdateMessage) {
	message := fmt.Sprintf("Notification for order %s: Status changed from '%s' to '%s' by %s at %s",
		statusUpdate.OrderNumber,
		statusUpdate.OldStatus,
		statusUpdate.NewStatus,
		statusUpdate.ChangedBy,
		statusUpdate.Timestamp.Format(time.RFC3339),
	)

	if !statusUpdate.EstimatedCompletion.IsZero() {
		message += fmt.Sprintf(". Estimated completion: %s",
			statusUpdate.EstimatedCompletion.Format(time.RFC3339))
	}

	fmt.Println(message)
}
