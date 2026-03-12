package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/BenedictKing/new_api_tools/internal/database"
	"github.com/BenedictKing/new_api_tools/internal/util"
)

// TopUpRecord represents a top-up record
type TopUpRecord struct {
	ID            int64   `json:"id" db:"id"`
	UserID        int64   `json:"user_id" db:"user_id"`
	Username      *string `json:"username" db:"username"`
	Amount        int64   `json:"amount" db:"amount"`
	Money         float64 `json:"money" db:"money"`
	TradeNo       string  `json:"trade_no" db:"trade_no"`
	PaymentMethod string  `json:"payment_method" db:"payment_method"`
	CreateTime    int64   `json:"create_time" db:"create_time"`
	CompleteTime  int64   `json:"complete_time" db:"complete_time"`
	Status        string  `json:"status" db:"status"`
}

// TopUpStatistics holds aggregate top-up statistics
type TopUpStatistics struct {
	TotalCount    int64   `json:"total_count"`
	TotalAmount   int64   `json:"total_amount"`
	TotalMoney    float64 `json:"total_money"`
	SuccessCount  int64   `json:"success_count"`
	SuccessAmount int64   `json:"success_amount"`
	SuccessMoney  float64 `json:"success_money"`
	PendingCount  int64   `json:"pending_count"`
	PendingAmount int64   `json:"pending_amount"`
	PendingMoney  float64 `json:"pending_money"`
	FailedCount   int64   `json:"failed_count"`
	FailedAmount  int64   `json:"failed_amount"`
	FailedMoney   float64 `json:"failed_money"`
}

// ListTopUpParams holds list query parameters
type ListTopUpParams struct {
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	UserID        *int64 `json:"user_id"`
	Status        string `json:"status"`
	PaymentMethod string `json:"payment_method"`
	TradeNo       string `json:"trade_no"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
}

// PaginatedTopUps holds paginated top-up results
type PaginatedTopUps struct {
	Items      []TopUpRecord `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// ListTopUpRecords lists top-up records with pagination and filtering
func ListTopUpRecords(params ListTopUpParams) (*PaginatedTopUps, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	db := database.GetManager()

	where := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.UserID != nil {
		where = append(where, fmt.Sprintf("t.user_id = %s", db.Placeholder(argIdx)))
		args = append(args, *params.UserID)
		argIdx++
	}

	if params.Status != "" {
		switch params.Status {
		case "success":
			where = append(where, "(LOWER(t.status) IN ('success', 'completed') OR t.status = '1')")
		case "failed":
			where = append(where, "(LOWER(t.status) IN ('failed', 'error') OR t.status = '-1')")
		case "pending":
			where = append(where, "(LOWER(t.status) NOT IN ('success', 'failed', 'completed', 'error') AND t.status NOT IN ('1', '-1'))")
		}
	}

	if params.PaymentMethod != "" {
		where = append(where, fmt.Sprintf("t.payment_method = %s", db.Placeholder(argIdx)))
		args = append(args, params.PaymentMethod)
		argIdx++
	}

	if params.TradeNo != "" {
		where = append(where, fmt.Sprintf("t.trade_no LIKE %s", db.Placeholder(argIdx)))
		args = append(args, "%"+params.TradeNo+"%")
		argIdx++
	}

	if params.StartDate != "" {
		ts, err := util.ParseDateToTimestampPublic(params.StartDate, false)
		if err == nil {
			where = append(where, fmt.Sprintf("t.create_time >= %s", db.Placeholder(argIdx)))
			args = append(args, ts)
			argIdx++
		}
	}

	if params.EndDate != "" {
		ts, err := util.ParseDateToTimestampPublic(params.EndDate, true)
		if err == nil {
			where = append(where, fmt.Sprintf("t.create_time <= %s", db.Placeholder(argIdx)))
			args = append(args, ts)
			argIdx++
		}
	}

	whereSQL := "1=1"
	if len(where) > 0 {
		whereSQL = strings.Join(where, " AND ")
	}

	// Count
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM top_ups t WHERE %s", whereSQL)
	countResult, err := db.Query(countSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}
	var total int64
	if len(countResult) > 0 {
		total = toInt64(countResult[0]["COUNT(*)"])
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	offset := (params.Page - 1) * params.PageSize

	// Select with user join
	selectSQL := fmt.Sprintf(`SELECT t.id, t.user_id, u.username, t.amount, t.money, COALESCE(t.trade_no,'') as trade_no, COALESCE(t.payment_method,'') as payment_method, COALESCE(t.create_time,0) as create_time, COALESCE(t.complete_time,0) as complete_time, COALESCE(t.status,'') as status FROM top_ups t LEFT JOIN users u ON t.user_id = u.id WHERE %s ORDER BY t.create_time DESC LIMIT %s OFFSET %s`,
		whereSQL, db.Placeholder(argIdx), db.Placeholder(argIdx+1))
	args = append(args, params.PageSize, offset)

	rows, err := db.Query(selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("select query failed: %w", err)
	}

	var items []TopUpRecord
	for _, row := range rows {
		item := TopUpRecord{
			ID:            toInt64(row["id"]),
			UserID:        toInt64(row["user_id"]),
			Amount:        toInt64(row["amount"]),
			Money:         toFloat64(row["money"]),
			TradeNo:       toString(row["trade_no"]),
			PaymentMethod: toString(row["payment_method"]),
			CreateTime:    toInt64(row["create_time"]),
			CompleteTime:  toInt64(row["complete_time"]),
			Status:        toString(row["status"]),
		}
		if username := row["username"]; username != nil {
			usernameStr := toString(username)
			item.Username = &usernameStr
		}
		items = append(items, item)
	}

	if items == nil {
		items = []TopUpRecord{}
	}

	return &PaginatedTopUps{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTopUpStatistics returns aggregate top-up statistics
func GetTopUpStatistics(startDate, endDate string) (*TopUpStatistics, error) {
	db := database.GetManager()

	where := []string{}
	args := []interface{}{}
	argIdx := 1

	if startDate != "" {
		ts, err := util.ParseDateToTimestampPublic(startDate, false)
		if err == nil {
			where = append(where, fmt.Sprintf("create_time >= %s", db.Placeholder(argIdx)))
			args = append(args, ts)
			argIdx++
		}
	}
	if endDate != "" {
		ts, err := util.ParseDateToTimestampPublic(endDate, true)
		if err == nil {
			where = append(where, fmt.Sprintf("create_time <= %s", db.Placeholder(argIdx)))
			args = append(args, ts)
			argIdx++
		}
	}

	whereSQL := "1=1"
	if len(where) > 0 {
		whereSQL = strings.Join(where, " AND ")
	}

	sql := fmt.Sprintf(`SELECT
		COUNT(*) as total_count,
		COALESCE(SUM(amount), 0) as total_amount,
		COALESCE(SUM(money), 0) as total_money,
		COALESCE(SUM(CASE WHEN LOWER(status) IN ('success', 'completed') OR status = '1' THEN 1 ELSE 0 END), 0) as success_count,
		COALESCE(SUM(CASE WHEN LOWER(status) IN ('success', 'completed') OR status = '1' THEN amount ELSE 0 END), 0) as success_amount,
		COALESCE(SUM(CASE WHEN LOWER(status) IN ('success', 'completed') OR status = '1' THEN money ELSE 0 END), 0) as success_money,
		COALESCE(SUM(CASE WHEN LOWER(status) IN ('failed', 'error') OR status = '-1' THEN 1 ELSE 0 END), 0) as failed_count,
		COALESCE(SUM(CASE WHEN LOWER(status) IN ('failed', 'error') OR status = '-1' THEN amount ELSE 0 END), 0) as failed_amount,
		COALESCE(SUM(CASE WHEN LOWER(status) IN ('failed', 'error') OR status = '-1' THEN money ELSE 0 END), 0) as failed_money
		FROM top_ups WHERE %s`, whereSQL)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("statistics query failed: %w", err)
	}
	if len(rows) == 0 {
		return &TopUpStatistics{}, nil
	}

	row := rows[0]
	totalCount := toInt64(row["total_count"])
	totalAmount := toInt64(row["total_amount"])
	totalMoney := toFloat64(row["total_money"])
	successCount := toInt64(row["success_count"])
	successAmount := toInt64(row["success_amount"])
	successMoney := toFloat64(row["success_money"])
	failedCount := toInt64(row["failed_count"])
	failedAmount := toInt64(row["failed_amount"])
	failedMoney := toFloat64(row["failed_money"])

	return &TopUpStatistics{
		TotalCount:    totalCount,
		TotalAmount:   totalAmount,
		TotalMoney:    totalMoney,
		SuccessCount:  successCount,
		SuccessAmount: successAmount,
		SuccessMoney:  successMoney,
		PendingCount:  totalCount - successCount - failedCount,
		PendingAmount: totalAmount - successAmount - failedAmount,
		PendingMoney:  totalMoney - successMoney - failedMoney,
		FailedCount:   failedCount,
		FailedAmount:  failedAmount,
		FailedMoney:   failedMoney,
	}, nil
}

// GetPaymentMethods returns distinct payment methods
func GetPaymentMethods() ([]string, error) {
	db := database.GetManager()
	rows, err := db.Query("SELECT DISTINCT payment_method FROM top_ups WHERE payment_method IS NOT NULL AND payment_method != '' ORDER BY payment_method")
	if err != nil {
		return nil, err
	}

	methods := []string{}
	for _, row := range rows {
		if method := toString(row["payment_method"]); method != "" {
			methods = append(methods, method)
		}
	}
	return methods, nil
}

// GetTopUpByID returns a single top-up record
func GetTopUpByID(id int64) (*TopUpRecord, error) {
	db := database.GetManager()
	sql := fmt.Sprintf(`SELECT t.id, t.user_id, u.username, t.amount, t.money, COALESCE(t.trade_no,'') as trade_no, COALESCE(t.payment_method,'') as payment_method, COALESCE(t.create_time,0) as create_time, COALESCE(t.complete_time,0) as complete_time, COALESCE(t.status,'') as status FROM top_ups t LEFT JOIN users u ON t.user_id = u.id WHERE t.id = %s`, db.Placeholder(1))

	rows, err := db.Query(sql, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("top-up record not found")
	}

	row := rows[0]
	rec := &TopUpRecord{
		ID:            toInt64(row["id"]),
		UserID:        toInt64(row["user_id"]),
		Amount:        toInt64(row["amount"]),
		Money:         toFloat64(row["money"]),
		TradeNo:       toString(row["trade_no"]),
		PaymentMethod: toString(row["payment_method"]),
		CreateTime:    toInt64(row["create_time"]),
		CompleteTime:  toInt64(row["complete_time"]),
		Status:        toString(row["status"]),
	}
	if username := row["username"]; username != nil {
		usernameStr := toString(username)
		rec.Username = &usernameStr
	}
	return rec, nil
}

// Unused import guard
var _ = time.Now
