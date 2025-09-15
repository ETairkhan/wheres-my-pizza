package validation

import (
	"errors"
	"regexp"
	"unicode/utf8"
	"wheres-my-pizza/pkg/models"
)

type OrderValidator struct{}

func NewOrderValidator() *OrderValidator {
	return &OrderValidator{}
}

func (v *OrderValidator) Validate(req *models.CreateOrderRequest) error {
	// Validate customer name
	if err := v.validateCustomerName(req.CustomerName); err != nil {
		return err
	}

	// Validate order type
	if err := v.validateOrderType(req.OrderType); err != nil {
		return err
	}

	// Validate items
	if err := v.validateItems(req.Items); err != nil {
		return err
	}

	// Validate conditional fields based on order type
	switch req.OrderType {
	case "dine_in":
		if req.TableNumber == nil {
			return errors.New("table_number is required for dine_in orders")
		}
		if *req.TableNumber < 1 || *req.TableNumber > 100 {
			return errors.New("table_number must be between 1 and 100")
		}
		if req.DeliveryAddress != nil {
			return errors.New("delivery_address must not be present for dine_in orders")
		}
	case "delivery":
		if req.DeliveryAddress == nil {
			return errors.New("delivery_address is required for delivery orders")
		}
		if utf8.RuneCountInString(*req.DeliveryAddress) < 10 {
			return errors.New("delivery_address must be at least 10 characters long")
		}
		if req.TableNumber != nil {
			return errors.New("table_number must not be present for delivery orders")
		}
	case "takeout":
		if req.TableNumber != nil {
			return errors.New("table_number must not be present for takeout orders")
		}
		if req.DeliveryAddress != nil {
			return errors.New("delivery_address must not be present for takeout orders")
		}
	}

	return nil
}

func (v *OrderValidator) validateCustomerName(name string) error {
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return errors.New("customer_name must be between 1 and 100 characters")
	}

	// Allow letters, spaces, hyphens, and apostrophes
	validName := regexp.MustCompile(`^[a-zA-Z\s\-']+$`)
	if !validName.MatchString(name) {
		return errors.New("customer_name contains invalid characters")
	}

	return nil
}

func (v *OrderValidator) validateOrderType(orderType string) error {
	switch orderType {
	case "dine_in", "takeout", "delivery":
		return nil
	default:
		return errors.New("order_type must be one of: dine_in, takeout, delivery")
	}
}

func (v *OrderValidator) validateItems(items []models.OrderItemRequest) error {
	if len(items) < 1 || len(items) > 20 {
		return errors.New("items must contain between 1 and 20 items")
	}

	for _, item := range items {
		if utf8.RuneCountInString(item.Name) < 1 || utf8.RuneCountInString(item.Name) > 50 {
			return errors.New("item name must be between 1 and 50 characters")
		}
		if item.Quantity < 1 || item.Quantity > 10 {
			return errors.New("item quantity must be between 1 and 10")
		}
		if item.Price < 0.01 || item.Price > 999.99 {
			return errors.New("item price must be between 0.01 and 999.99")
		}
	}

	return nil
}
