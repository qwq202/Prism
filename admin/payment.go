package admin

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"math"
	"time"
)

type PaymentOrder struct {
	Id        int64     `json:"id"`
	UserId    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Type      string    `json:"type"`
	Service   string    `json:"service"`
	Amount    float64   `json:"amount"`
	OrderId   string    `json:"order_id"`
	Name      string    `json:"name"`
	Device    string    `json:"device"`
	State     bool      `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaymentPaginationForm struct {
	Status  bool           `json:"status"`
	Message string         `json:"message,omitempty"`
	Total   int            `json:"total"`
	Data    []PaymentOrder `json:"data"`
}

func getPaymentOrdersForm(db *sql.DB, page int64, search string) PaymentPaginationForm {
	if page < 0 {
		page = 0
	}

	var total int64
	likeSearch := "%" + search + "%"
	if err := globals.QueryRowDb(db, `
		SELECT COUNT(*)
		FROM payment_orders p
		LEFT JOIN auth a ON a.id = p.user_id
		WHERE p.order_id LIKE ? OR p.username LIKE ? OR a.username LIKE ?
	`, likeSearch, likeSearch, likeSearch).Scan(&total); err != nil {
		return PaymentPaginationForm{Status: false, Message: err.Error()}
	}

	rows, err := globals.QueryDb(db, `
		SELECT p.id, p.user_id, COALESCE(a.username, p.username, ''), 
		       COALESCE(p.type, ''), COALESCE(p.service, ''),
		       p.amount, p.order_id, COALESCE(p.name, ''), COALESCE(p.device, ''),
		       p.state, p.created_at, p.updated_at
		FROM payment_orders p
		LEFT JOIN auth a ON a.id = p.user_id
		WHERE p.order_id LIKE ? OR p.username LIKE ? OR a.username LIKE ?
		ORDER BY p.id DESC
		LIMIT ? OFFSET ?
	`, likeSearch, likeSearch, likeSearch, pagination, page*pagination)
	if err != nil {
		return PaymentPaginationForm{Status: false, Message: err.Error()}
	}
	defer rows.Close()

	var orders []PaymentOrder
	for rows.Next() {
		var o PaymentOrder
		var createdAt, updatedAt []uint8
		if err := rows.Scan(
			&o.Id, &o.UserId, &o.Username, &o.Type, &o.Service,
			&o.Amount, &o.OrderId, &o.Name, &o.Device, &o.State,
			&createdAt, &updatedAt,
		); err != nil {
			return PaymentPaginationForm{Status: false, Message: err.Error()}
		}
		if t := utils.ConvertTime(createdAt); t != nil {
			o.CreatedAt = *t
		}
		if t := utils.ConvertTime(updatedAt); t != nil {
			o.UpdatedAt = *t
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return PaymentPaginationForm{Status: false, Message: err.Error()}
	}

	if orders == nil {
		orders = []PaymentOrder{}
	}

	return PaymentPaginationForm{
		Status: true,
		Total:  int(math.Ceil(float64(total) / float64(pagination))),
		Data:   orders,
	}
}

func recheckPaymentOrder(db *sql.DB, orderId string) (bool, bool, error) {
	var state bool
	err := globals.QueryRowDb(db, `
		SELECT state FROM payment_orders WHERE order_id = ?
	`, orderId).Scan(&state)
	if err != nil {
		return false, false, fmt.Errorf("order not found")
	}

	return true, state, nil
}
