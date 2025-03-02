package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/assylzhan-a/outboxie/internal/example/model"
	"github.com/assylzhan-a/outboxie/internal/example/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var order model.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	createdOrder, err := h.orderService.CreateOrder(ctx, &order)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create order: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdOrder)
}
