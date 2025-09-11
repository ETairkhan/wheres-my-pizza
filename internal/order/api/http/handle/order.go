package handle

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/order/app/core"
	"wheres-my-pizza/internal/order/app/services"
	"wheres-my-pizza/internal/order/domain/dto"
)

type OrderHandler struct {
	orderService *services.OrderService
	mylog        logger.Logger
}

func NewOrderHandler(orderService *services.OrderService, mylog logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		mylog:        mylog,
	}
}

func (oh *OrderHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var order dto.OrderRequest

		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			oh.mylog.Action("parse_failed").Error("Failed to parse order", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		oh.mylog.Action("parse_completed").Info("order successfully parsed")

		if err := oh.orderService.ValidateOrder(order); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		oh.mylog.Action("received").Debug("Received order info", "customer_name", order.CustomerName, "delivery_address", order.DeliveryAddress, "number_of_items", len(order.Items))

		ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime*time.Second)
		defer cancel()

		newOrder, err := oh.orderService.Create(ctx, order)
		if err != nil {
			if errors.Is(err, core.ErrDBConn) {
				jsonError(w, http.StatusInternalServerError, err)
				return
			}
			if errors.Is(err, core.ErrMaxConcurentExceeded) {
				jsonError(w, http.StatusInternalServerError, err)
				return
			}
			jsonError(w, http.StatusInternalServerError, errors.New("failed to add order"))
			return
		}

		oh.mylog.Action("published").Debug("Published order info", "order_number", newOrder.OrderNumber, "status", newOrder.Status, "total_amount", newOrder.TotalAmount)

		resp := dto.OrderResponse{
			OrderNumber: newOrder.OrderNumber,
			Status:      newOrder.Status,
			TotalAmount: newOrder.TotalAmount,
		}
		jsonResponse(w, http.StatusOK, resp)
		oh.mylog.Action("completed").Info("New order is added to db successfully!")
	}
}
