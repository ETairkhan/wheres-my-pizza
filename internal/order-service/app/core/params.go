package core

type OrderParams struct {
	Port          int
	MaxConcurrent int
}

const (
	// Constraints for customer name
	MinCustomerNameLen = 1
	MaxCustomerNameLen = 100

	AllowedSpecialCharacters = "-' "

	// Constraints for items
	MinItems = 1
	MaxItems = 20

	MinItemNameLen = 1
	MaxItemNameLen = 50

	MinItemQuantity = 1
	MaxItemQuantity = 10

	MinItemPrice = 0.01
	MaxItemPrice = 999.99

	// Constraints for conditional validation
	MinTableNumber = 1
	MaxTableNumber = 100

	MinDeliveryAddressLen = 10
	MaxDeliveryAddressLen = 100

	// Priority Levels
	MaxPriority = 10
	MedPriority = 5
	MinPriority = 1

	// in seconds for db response
	WaitTime = 20

	DefaultOrderStatus = "received"
	DefaultProcessedBy = "order-service"

	RMQReconnectionInterval = 5
)

var AllowedTypes = map[string]bool{
	"dine_in":  true,
	"takeout":  true,
	"delivery": true,
}
