package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/order/app/core"
	"wheres-my-pizza/internal/order/domain/dto"
	"wheres-my-pizza/internal/order/domain/models"
)

type OrderService struct {
	ctx           context.Context
	orderRepo     core.IOrderRepo
	messageBroker core.IRabbitMQ
	mylog         logger.Logger
}

func NewOrderService(
	ctx context.Context,
	orderRepo core.IOrderRepo,
	messageBroker core.IRabbitMQ,
	mylogger logger.Logger,
) *OrderService {
	return &OrderService{
		ctx:           ctx,
		orderRepo:     orderRepo,
		mylog:         mylogger,
		messageBroker: messageBroker,
	}
}

func (os *OrderService) Create(ctx context.Context, order dto.OrderRequest) (models.Order, error) {
	mylog := os.mylog.Action("create order")
	mylog.Info("Preparing order for DB insert")

	// calculate total
	total := 0.0
	for _, item := range order.Items {
		total += float64(item.Quantity) * item.Price
	}
	order.TotalAmount = total

	// set priority
	if total > 100 {
		order.Priority = core.MaxPriority
	} else if total >= 50 {
		order.Priority = core.MedPriority
	} else {
		order.Priority = core.MinPriority
	}

	// add order to db
	newOrder, err := os.orderRepo.Create(ctx, order)
	if err != nil {
		if errors.Is(err, core.ErrDBConn) {
			mylog.Error("Failed to connect to db", err)
			return models.Order{}, fmt.Errorf("cannot connect to db: %w", err)
		}
		if errors.Is(err, core.ErrMaxConcurentExceeded) {
			mylog.Warn("Failed to create, to many orders right now, boy")
			return models.Order{}, err
		}
		mylog.Error("Failed to save order record in db", err)
		return models.Order{}, fmt.Errorf("cannot save order in db: %w", err)
	}
	order.OrderNumber = newOrder.OrderNumber

	if err := os.messageBroker.PushMessage(os.ctx, order); err != nil {
		mylog.Error("Failed to publish message", err)
		return models.Order{}, fmt.Errorf("cannot send message to broker: %w", err)
	}

	mylog.Info("Order created successfully")
	return newOrder, nil
}

// ValidateOrder validates order json against predefined rules
func (os *OrderService) ValidateOrder(order dto.OrderRequest) error {
	os.mylog.Action("validation_started").Info("Validating order request")

	if err := os.validateCustomerName(order.CustomerName); err != nil {
		return fmt.Errorf("invalid customer name: %v", err)
	}

	if err := os.validateOrderType(order); err != nil {
		return fmt.Errorf("invalid order type: %v", err)
	}

	if err := os.validateOrderItems(order.Items); err != nil {
		return fmt.Errorf("invalid order items: %v", err)
	}

	os.mylog.Action("validation_completed").Info("Order successfully validated")
	return nil
}

func (os *OrderService) validateCustomerName(customerName string) error {
	if customerName == "" {
		return core.ErrFieldIsEmpty
	}

	customerNameLen := len(customerName)
	if customerNameLen < core.MinCustomerNameLen || customerNameLen > core.MaxCustomerNameLen {
		return fmt.Errorf("must be in range [%d, %d]", core.MinCustomerNameLen, core.MaxCustomerNameLen)
	}

	for _, r := range customerName {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || strings.ContainsRune(core.AllowedSpecialCharacters, r) {
			continue
		} else {
			return fmt.Errorf("must not contain special characters other than `%s`: %s", core.AllowedSpecialCharacters, customerName)
		}
	}

	return nil
}

func (os *OrderService) validateOrderType(order dto.OrderRequest) error {
	// Checking Order Type
	if order.Type == "" {
		return core.ErrFieldIsEmpty
	}
	if !core.AllowedTypes[order.Type] {
		return fmt.Errorf("undefined type: %s", order.Type)
	}

	if err := os.validateOrderTypeParams(order); err != nil {
		return fmt.Errorf("invalid %s parameter: %v", order.Type, err)
	}

	return nil
}

func (os *OrderService) validateOrderTypeParams(order dto.OrderRequest) error {
	// Conditional Validation
	switch order.Type {
	case "dine_in":
		order.DeliveryAddress = ""

		if order.TableNumber == 0 {
			return core.ErrFieldIsEmpty
		}

		if order.TableNumber < core.MinTableNumber || order.TableNumber > core.MaxTableNumber {
			return fmt.Errorf("table number: %d, must be in range [%d, %d]", order.TableNumber, core.MinTableNumber, core.MaxTableNumber)
		}

	case "delivery":
		order.TableNumber = 0

		if order.DeliveryAddress == "" {
			return core.ErrFieldIsEmpty
		}

		deliveryAddressLen := len(order.DeliveryAddress)
		if deliveryAddressLen < core.MinDeliveryAddressLen || deliveryAddressLen > core.MaxDeliveryAddressLen {
			return fmt.Errorf("address length: %d, must be in range [%d, %d]", deliveryAddressLen, core.MinDeliveryAddressLen, core.MaxDeliveryAddressLen)
		}
	case "takeout":
		order.DeliveryAddress = ""
		order.TableNumber = 0
	}
	return nil
}

func (os *OrderService) validateOrderItems(items []dto.Item) error {
	// Checking Items amount
	itemsLen := len(items)
	if itemsLen == 0 {
		return core.ErrFieldIsEmpty
	}
	if itemsLen < core.MinItems || itemsLen > core.MaxItems {
		return fmt.Errorf("amount of items: %d, must be in range [%d, %d]", itemsLen, core.MinItems, core.MaxItems)
	}

	for i, item := range items {
		itemNameLen := len(item.Name)
		// Check name
		if itemNameLen < core.MinItemNameLen || itemNameLen > core.MaxItemNameLen {
			return fmt.Errorf("item %d: name len: %d, must be in range [%d, %d]", i+1, itemNameLen, core.MinItemNameLen, core.MaxItemNameLen)
		}
		// Check quantity
		if item.Quantity < core.MinItemQuantity || item.Quantity > core.MaxItemQuantity {
			return fmt.Errorf("item %d: quantity: %d, must be in range [%d, %d]", i+1, item.Quantity, core.MinItemQuantity, core.MaxItemQuantity)
		}
		// Check price
		if item.Price < core.MinItemPrice || item.Price > core.MaxItemPrice {
			return fmt.Errorf("item %d: item price: %f, must be in range [%f, %f]", i+1, item.Price, core.MinItemPrice, core.MaxItemPrice)
		}
	}

	return nil
}
