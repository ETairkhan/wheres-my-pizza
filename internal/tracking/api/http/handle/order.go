package handle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"where-is-my-pizza/internal/mylogger"
	"where-is-my-pizza/internal/tracking/app/core"
	"where-is-my-pizza/internal/tracking/app/services"
)

type OrderHandler struct {
	orderService *services.OrderService
	mylog        mylogger.Logger
}

func NewOrderHandler(orderService *services.OrderService, mylog mylogger.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		mylog:        mylog,
	}
}

func (oh *OrderHandler) GetStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderNum := r.PathValue("order_number")

		ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime*time.Second)
		defer cancel()

		order, err := oh.orderService.GetStatus(ctx, orderNum)
		if err != nil {
			if errors.Is(err, core.ErrOrderNotFound) {
				jsonError(w, http.StatusNotFound, core.ErrOrderNotFound)
				return
			}
			jsonError(w, http.StatusBadRequest, fmt.Errorf("failed to get order status: %v", err))
			return
		}
		estimatedCompletion := oh.orderService.GetEstimatedCompletion(order.CreatedAt.UTC(), order.Type)

		type resp struct {
			OrderNumber         string `json:"order_number"`
			CurrentStatus       string `json:"current_status"`
			UpdatedAt           string `json:"updated_at"`
			EstimatedCompletion string `json:"estimated_completion"`
			ProcessedBy         string `json:"processed_by"`
		}
		orderStatus := resp{
			OrderNumber:         order.OrderNumber,
			CurrentStatus:       order.Status,
			UpdatedAt:           order.UpdatedAt.UTC().Format(time.RFC3339),
			EstimatedCompletion: estimatedCompletion.UTC().Format(time.RFC3339),
			ProcessedBy:         order.ProcessedBy,
		}

		jsonResponse(w, http.StatusOK, orderStatus)
	}
}

func (oh *OrderHandler) GetHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderNum := r.PathValue("order_number")

		ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime*time.Second)
		defer cancel()

		osls, err := oh.orderService.GetHistory(ctx, orderNum)
		if err != nil {
			if errors.Is(err, core.ErrOrderNotFound) {
				jsonError(w, http.StatusNotFound, core.ErrOrderNotFound)
				return
			}
			jsonError(w, http.StatusInternalServerError, fmt.Errorf("failed to get order history: %v", err))
			return
		}

		type resp struct {
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
			ChangedBy string `json:"changed_by"`
		}

		var history []resp
		for _, osl := range osls {
			history = append(history,
				resp{
					Status:    osl.Status,
					Timestamp: osl.ChangedAt.UTC().Format(time.RFC3339),
					ChangedBy: osl.ChangedBy,
				},
			)
		}

		jsonResponse(w, http.StatusOK, history)
	}
}
