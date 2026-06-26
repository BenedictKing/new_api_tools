package service

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	UptimeKumaStatusDown        = 0
	UptimeKumaStatusUp          = 1
	UptimeKumaStatusPending     = 2
	UptimeKumaStatusMaintenance = 3
)

type UptimeKumaService struct {
	modelStatusService *ModelStatusService
}

func NewUptimeKumaService() *UptimeKumaService {
	return &UptimeKumaService{
		modelStatusService: NewModelStatusService(),
	}
}

type StatusPageConfig struct {
	Slug                string `json:"slug"`
	Title               string `json:"title"`
	Description         string `json:"description,omitempty"`
	Icon                string `json:"icon"`
	Theme               string `json:"theme"`
	Published           bool   `json:"published"`
	ShowTags            bool   `json:"showTags"`
	AutoRefreshInterval int    `json:"autoRefreshInterval"`
}

type MonitorItem struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	SendURL int    `json:"sendUrl"`
}

type MonitorGroup struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	Weight      int           `json:"weight"`
	MonitorList []MonitorItem `json:"monitorList"`
}

type StatusPageData struct {
	Config          StatusPageConfig `json:"config"`
	Incident        interface{}      `json:"incident"`
	PublicGroupList []MonitorGroup   `json:"publicGroupList"`
	MaintenanceList []interface{}    `json:"maintenanceList"`
}

type HeartbeatItem struct {
	Status int    `json:"status"`
	Time   string `json:"time"`
	Msg    string `json:"msg"`
	Ping   *int   `json:"ping"`
}

type HeartbeatData struct {
	HeartbeatList map[string][]HeartbeatItem `json:"heartbeatList"`
	UptimeList    map[string]float64         `json:"uptimeList"`
}

type BadgeData struct {
	Label  string `json:"label"`
	Status string `json:"status"`
	Color  string `json:"color"`
}

type SummaryData struct {
	Success         bool    `json:"success"`
	Status          int     `json:"status"`
	StatusText      string  `json:"status_text"`
	Uptime          float64 `json:"uptime"`
	TotalMonitors   int     `json:"total_monitors"`
	MonitorsUp      int     `json:"monitors_up"`
	MonitorsDown    int     `json:"monitors_down"`
	MonitorsPending int     `json:"monitors_pending"`
	LastUpdated     string  `json:"last_updated"`
}

func modelNameToUptimeKumaID(modelName string) int {
	hash := md5.Sum([]byte(modelName))
	id := binary.BigEndian.Uint32(hash[:4])
	return int(id % 1000000000)
}

func mapStatusToUptimeKuma(successRate float64, totalRequests int64) int {
	if totalRequests == 0 || successRate >= 95 {
		return UptimeKumaStatusUp
	}
	if successRate >= 80 {
		return UptimeKumaStatusPending
	}
	return UptimeKumaStatusDown
}

func (s *UptimeKumaService) GetStatusPageConfig(slug, window string) (*StatusPageData, error) {
	statuses, err := s.allModelStatuses(window)
	if err != nil {
		return nil, err
	}

	monitorList := make([]MonitorItem, 0, len(statuses))
	for _, status := range statuses {
		modelName := toString(status["model_name"])
		displayName := toString(status["display_name"])
		if displayName == "" {
			displayName = modelName
		}
		monitorList = append(monitorList, MonitorItem{
			ID:      modelNameToUptimeKumaID(modelName),
			Name:    displayName,
			Type:    "http",
			SendURL: 0,
		})
	}

	return &StatusPageData{
		Config: StatusPageConfig{
			Slug:                slug,
			Title:               "Model Status",
			Description:         "AI Model Health Status",
			Icon:                "/tool.svg",
			Theme:               "auto",
			Published:           true,
			ShowTags:            false,
			AutoRefreshInterval: 60,
		},
		Incident: nil,
		PublicGroupList: []MonitorGroup{{
			ID:          1,
			Name:        "AI Models",
			Weight:      1,
			MonitorList: monitorList,
		}},
		MaintenanceList: []interface{}{},
	}, nil
}

func (s *UptimeKumaService) GetHeartbeatData(slug, window string) (*HeartbeatData, error) {
	statuses, err := s.allModelStatuses(window)
	if err != nil {
		return nil, err
	}

	heartbeatList := make(map[string][]HeartbeatItem, len(statuses))
	uptimeList := make(map[string]float64, len(statuses))

	for _, status := range statuses {
		modelName := toString(status["model_name"])
		monitorID := fmt.Sprintf("%d", modelNameToUptimeKumaID(modelName))

		slots := readStatusSlots(status["slot_data"])
		heartbeats := make([]HeartbeatItem, 0, len(slots))
		for _, slot := range slots {
			successRate := toFloat64(slot["success_rate"])
			totalRequests := toInt64(slot["total_requests"])
			successCount := toInt64(slot["success_count"])
			endTime := toInt64(slot["end_time"])

			msg := ""
			if totalRequests > 0 {
				msg = fmt.Sprintf("%d/%d (%.1f%%)", successCount, totalRequests, successRate)
			}

			heartbeats = append(heartbeats, HeartbeatItem{
				Status: mapStatusToUptimeKuma(successRate, totalRequests),
				Time:   time.Unix(endTime, 0).UTC().Format("2006-01-02 15:04:05"),
				Msg:    msg,
				Ping:   nil,
			})
		}

		heartbeatList[monitorID] = heartbeats
		uptimeList[monitorID+"_24"] = toFloat64(status["success_rate"]) / 100.0
	}

	return &HeartbeatData{HeartbeatList: heartbeatList, UptimeList: uptimeList}, nil
}

func (s *UptimeKumaService) GetBadgeData(slug, window, label string) (*BadgeData, error) {
	statuses, err := s.allModelStatuses(window)
	if err != nil {
		return nil, err
	}

	hasUp := false
	hasDown := false
	for _, status := range statuses {
		current := mapStatusToUptimeKuma(toFloat64(status["success_rate"]), toInt64(status["total_requests"]))
		if current == UptimeKumaStatusUp {
			hasUp = true
		}
		if current == UptimeKumaStatusDown {
			hasDown = true
		}
	}

	badgeStatus := "N/A"
	color := "#808080"
	switch {
	case hasUp && !hasDown:
		badgeStatus = "Up"
		color = "#4CAF50"
	case hasUp && hasDown:
		badgeStatus = "Degraded"
		color = "#F6BE00"
	case hasDown:
		badgeStatus = "Down"
		color = "#DC3545"
	}

	return &BadgeData{Label: label, Status: badgeStatus, Color: color}, nil
}

func (s *UptimeKumaService) GetSummaryData(slug, window string) (*SummaryData, error) {
	statuses, err := s.allModelStatuses(window)
	if err != nil {
		return nil, err
	}

	var monitorsUp, monitorsDown, monitorsPending int
	var totalUptime float64
	for _, status := range statuses {
		current := mapStatusToUptimeKuma(toFloat64(status["success_rate"]), toInt64(status["total_requests"]))
		switch current {
		case UptimeKumaStatusUp:
			monitorsUp++
		case UptimeKumaStatusDown:
			monitorsDown++
		default:
			monitorsPending++
		}
		totalUptime += toFloat64(status["success_rate"])
	}

	totalMonitors := len(statuses)
	overallUptime := 100.0
	if totalMonitors > 0 {
		overallUptime = totalUptime / float64(totalMonitors)
	}

	overallStatus := UptimeKumaStatusUp
	statusText := "UP"
	if monitorsDown > 0 {
		overallStatus = UptimeKumaStatusDown
		statusText = "DOWN"
	} else if monitorsPending > 0 {
		overallStatus = UptimeKumaStatusPending
		statusText = "PENDING"
	}

	return &SummaryData{
		Success:         true,
		Status:          overallStatus,
		StatusText:      statusText,
		Uptime:          overallUptime,
		TotalMonitors:   totalMonitors,
		MonitorsUp:      monitorsUp,
		MonitorsDown:    monitorsDown,
		MonitorsPending: monitorsPending,
		LastUpdated:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *UptimeKumaService) allModelStatuses(window string) ([]map[string]interface{}, error) {
	models, err := s.modelStatusService.GetAvailableModels()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(models))
	for _, model := range models {
		name := toString(model["model_name"])
		if name != "" {
			names = append(names, name)
		}
	}
	return s.modelStatusService.GetMultipleModelsStatus(names, window)
}

func readStatusSlots(raw interface{}) []map[string]interface{} {
	switch slots := raw.(type) {
	case []map[string]interface{}:
		return slots
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(slots))
		for _, item := range slots {
			if slot, ok := item.(map[string]interface{}); ok {
				result = append(result, slot)
			}
		}
		return result
	default:
		return []map[string]interface{}{}
	}
}
