package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/new-api-tools/backend/internal/database"
	"github.com/new-api-tools/backend/internal/logger"
	"github.com/new-api-tools/backend/internal/service"
)

// InitDefaultTasks registers background tasks used by backend.
func InitDefaultTasks() (*Manager, error) {
	m := GetManager()
	registrations := []*Task{
		{
			Name:         "index_ensure",
			InitialDelay: 2 * time.Second,
			Interval:     24 * time.Hour,
			Timeout:      30 * time.Minute,
			Handler:      indexEnsureTask,
		},
		{
			Name:         "ip_recording_enforce",
			InitialDelay: 30 * time.Second,
			Interval:     10 * time.Minute,
			Timeout:      time.Minute,
			Handler:      ipRecordingEnforceTask,
		},
		{
			Name:         "abuse_broadcast_sync",
			InitialDelay: 20 * time.Second,
			Interval:     time.Minute,
			Timeout:      45 * time.Second,
			Handler:      abuseBroadcastSyncTask,
			NextInterval: abuseBroadcastNextInterval,
		},
		{
			Name:         "model_status_available_models_refresh",
			InitialDelay: 15 * time.Second,
			Interval:     10 * time.Minute,
			Timeout:      2 * time.Minute,
			Handler:      modelStatusAvailableModelsRefreshTask,
		},
		{
			Name:         "model_status_token_groups_refresh",
			InitialDelay: 20 * time.Second,
			Interval:     10 * time.Minute,
			Timeout:      2 * time.Minute,
			Handler:      modelStatusTokenGroupsRefreshTask,
		},
		{
			Name:         "analytics_summary_warmup",
			InitialDelay: 45 * time.Second,
			Interval:     10 * time.Minute,
			Timeout:      3 * time.Minute,
			Handler:      analyticsSummaryWarmupTask,
		},
		{
			Name:         "ai_ban_suspicious_readonly_scan",
			InitialDelay: 2 * time.Minute,
			Interval:     30 * time.Minute,
			Timeout:      2 * time.Minute,
			Handler:      aiBanSuspiciousReadonlyScanTask,
		},
		{
			Name:         "warmup_ready_guard",
			InitialDelay: 60 * time.Second,
			Interval:     24 * time.Hour,
			Timeout:      5 * time.Second,
			Handler:      warmupReadyGuardTask,
		},
	}

	for _, task := range registrations {
		if err := m.Register(task); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func indexEnsureTask(ctx context.Context) error {
	_ = ctx
	database.Get().EnsureIndexes(true, 500*time.Millisecond)
	return nil
}

func ipRecordingEnforceTask(ctx context.Context) error {
	_ = ctx
	svc := service.NewIPMonitoringService()
	stats, err := svc.GetIPStats()
	if err != nil {
		return fmt.Errorf("获取 IP 记录状态失败: %w", err)
	}
	disabledCount := toInt64(stats["disabled_count"])
	if disabledCount == 0 {
		logger.L.Debug("所有用户已开启 IP 记录，无需操作", logger.CatTask)
		return nil
	}
	logger.L.Task(fmt.Sprintf("检测到 %d 个用户关闭了 IP 记录，正在强制开启", disabledCount))
	result, err := svc.EnableAllIPRecording()
	if err != nil {
		return fmt.Errorf("强制开启 IP 记录失败: %w", err)
	}
	logger.L.Success(fmt.Sprintf("[IP记录] %s", result["message"]))
	return nil
}

func abuseBroadcastNextInterval(ctx context.Context) (time.Duration, error) {
	settings, err := service.NewAbuseBroadcastService().GetSettings(ctx)
	if err != nil {
		return time.Minute, err
	}
	if !settings.Enabled {
		return time.Minute, nil
	}
	seconds := settings.PullIntervalSeconds
	if seconds <= 0 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second, nil
}

func abuseBroadcastSyncTask(ctx context.Context) error {
	settings, err := service.NewAbuseBroadcastService().GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("读取违规广播配置失败: %w", err)
	}
	if !settings.Enabled {
		return nil
	}
	result, err := service.NewAbuseBroadcastService().SyncOnce(ctx)
	if err != nil {
		return fmt.Errorf("违规广播同步失败: %w", err)
	}
	if result.PulledEvents > 0 {
		logger.L.Success(fmt.Sprintf("[违规广播] 已同步 %d 个事件，写入 %d 条通报，cursor=%d",
			result.PulledEvents, result.StoredReports, result.NextCursor))
	}
	return nil
}

func modelStatusAvailableModelsRefreshTask(ctx context.Context) error {
	_ = ctx
	GetManager().warmup.MarkStep("模型列表缓存", "running", "正在刷新可用模型列表")
	models, err := service.NewModelStatusService().GetAvailableModels("all", true)
	if err != nil {
		GetManager().warmup.MarkStep("模型列表缓存", "error", err.Error())
		return fmt.Errorf("刷新可用模型列表失败: %w", err)
	}
	GetManager().warmup.MarkStep("模型列表缓存", "success", fmt.Sprintf("已缓存 %d 个模型", len(models)))
	return nil
}

func modelStatusTokenGroupsRefreshTask(ctx context.Context) error {
	_ = ctx
	GetManager().warmup.MarkStep("Token 分组缓存", "running", "正在刷新 Token 分组")
	groups, err := service.NewModelStatusService().GetTokenGroups()
	if err != nil {
		GetManager().warmup.MarkStep("Token 分组缓存", "error", err.Error())
		return fmt.Errorf("刷新 Token 分组失败: %w", err)
	}
	GetManager().warmup.MarkStep("Token 分组缓存", "success", fmt.Sprintf("已缓存 %d 个分组", len(groups)))
	return nil
}

func analyticsSummaryWarmupTask(ctx context.Context) error {
	_ = ctx
	GetManager().warmup.MarkStep("Analytics Summary 缓存", "running", "正在预热统计摘要")
	_, err := service.NewLogAnalyticsService().GetSummary()
	if err != nil {
		GetManager().warmup.MarkStep("Analytics Summary 缓存", "error", err.Error())
		return fmt.Errorf("预热统计摘要失败: %w", err)
	}
	GetManager().warmup.MarkStep("Analytics Summary 缓存", "success", "统计摘要已预热")
	GetManager().warmup.Ready("系统已就绪")
	return nil
}

func aiBanSuspiciousReadonlyScanTask(ctx context.Context) error {
	_ = ctx
	users, err := service.NewAIAutoBanService().GetSuspiciousUsers("24h", 100)
	if err != nil {
		return fmt.Errorf("AI Ban 可疑用户只读扫描失败: %w", err)
	}
	logger.L.Task(fmt.Sprintf("AI Ban 可疑用户只读扫描完成，发现 %d 个候选用户", len(users)))
	return nil
}

func warmupReadyGuardTask(ctx context.Context) error {
	_ = ctx
	status := GetManager().warmup.Status()
	if status.Status != "ready" {
		GetManager().warmup.Ready("系统已就绪，部分预热任务将在后台继续重试")
	}
	return nil
}

func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}
