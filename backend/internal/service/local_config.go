package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/new-api-tools/backend/internal/database"
)

func loadLocalConfig(key string, dest interface{}) bool {
	store := database.GetLocalStore()
	if store == nil {
		return false
	}
	value, found, err := store.GetConfig(context.Background(), key)
	if err != nil || !found {
		return false
	}
	return json.Unmarshal([]byte(value), dest) == nil
}

func saveLocalConfig(key string, value interface{}) error {
	store := database.GetLocalStore()
	if store == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return store.SetConfig(context.Background(), key, string(data))
}

func loadLocalList(key string) []map[string]interface{} {
	var items []map[string]interface{}
	if loadLocalConfig(key, &items) {
		return items
	}
	return []map[string]interface{}{}
}

func saveLocalList(key string, items []map[string]interface{}, limit int) error {
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return saveLocalConfig(key, items)
}

func prependLocalList(key string, entries []map[string]interface{}, limit int) error {
	if len(entries) == 0 {
		return nil
	}
	items := loadLocalList(key)
	merged := make([]map[string]interface{}, 0, len(entries)+len(items))
	merged = append(merged, entries...)
	merged = append(merged, items...)
	return saveLocalList(key, merged, limit)
}

func loadLocalInt64List(key string) []int64 {
	var ints []int64
	if loadLocalConfig(key, &ints) {
		return ints
	}

	var raw []interface{}
	if !loadLocalConfig(key, &raw) {
		return []int64{}
	}

	ints = make([]int64, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case float64:
			ints = append(ints, int64(v))
		case int64:
			ints = append(ints, v)
		case int:
			ints = append(ints, int64(v))
		case json.Number:
			if n, err := v.Int64(); err == nil {
				ints = append(ints, n)
			}
		case string:
			var n int64
			if _, err := fmt.Sscan(v, &n); err == nil {
				ints = append(ints, n)
			}
		}
	}
	return ints
}
