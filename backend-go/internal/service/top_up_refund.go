package service

import (
	"fmt"
	"strings"

	"github.com/BenedictKing/new_api_tools/internal/database"
	"github.com/BenedictKing/new_api_tools/internal/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RefundTopUp 退款：减少用户 quota，并将 top_ups 状态标记为 refunded。
// 约定：
// - 仅允许对“成功/已完成”的 top_up 退款；
// - 若用户 quota 不足以扣减，则扣到 0（不允许负数）；
// - 使用事务保证 top_ups 与 users 同步。
func RefundTopUp(topUpID int64) error {
	db := database.GetMainDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 读取 top_up 记录
	var rec struct {
		ID     int64
		UserID int64
		Amount int64
		Status string
	}
	if err := db.Raw("SELECT id, user_id, amount, COALESCE(status,'') as status FROM top_ups WHERE id = ?", topUpID).Scan(&rec).Error; err != nil {
		return err
	}
	if rec.ID == 0 {
		return fmt.Errorf("top-up record not found")
	}

	statusLower := strings.ToLower(rec.Status)
	if statusLower == "refunded" {
		return nil
	}
	if !(statusLower == "success" || statusLower == "completed" || rec.Status == "1") {
		return fmt.Errorf("top-up status not refundable: %s", rec.Status)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1) 锁定并读取用户 quota
		var quota int64
		if err := tx.Raw("SELECT quota FROM users WHERE id = ? FOR UPDATE", rec.UserID).Scan(&quota).Error; err != nil {
			return err
		}

		// 2) 计算扣减后的 quota（不允许负数）
		newQuota := quota - rec.Amount
		if newQuota < 0 {
			newQuota = 0
		}

		// 3) 更新用户 quota
		if err := tx.Exec("UPDATE users SET quota = ? WHERE id = ?", newQuota, rec.UserID).Error; err != nil {
			return err
		}

		// 4) 更新 top_ups 状态
		if err := tx.Exec("UPDATE top_ups SET status = 'refunded' WHERE id = ?", rec.ID).Error; err != nil {
			return err
		}

		logger.Info("TopUp refund completed",
			zap.Int64("top_up_id", rec.ID),
			zap.Int64("user_id", rec.UserID),
			zap.Int64("amount", rec.Amount),
			zap.Int64("quota_before", quota),
			zap.Int64("quota_after", newQuota),
		)
		return nil
	})
}
