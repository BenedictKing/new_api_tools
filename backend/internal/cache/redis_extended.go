package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Extended Redis operations for advanced caching features
// This file provides additional Redis operations beyond the basic Manager

var (
	extCtx = context.Background()
)

// ErrCacheMiss indicates cache key not found
var ErrCacheMiss = fmt.Errorf("cache miss")

// Exists checks if a cache key exists
func Exists(key string) (bool, error) {
	if !Available() {
		return false, nil
	}
	count, err := Get().rdb.Exists(extCtx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Expire sets expiration time for a key
func Expire(key string, ttl time.Duration) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.Expire(extCtx, key, ttl).Err()
}

// TTL gets remaining TTL for a key
func TTL(key string) (time.Duration, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}
	return Get().rdb.TTL(extCtx, key).Result()
}

// Keys gets keys matching pattern (use ScanKeys for production)
// WARNING: This uses KEYS command which may block Redis on large datasets
func Keys(pattern string) ([]string, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.Keys(extCtx, pattern).Result()
}

// ScanKeys iterates keys using SCAN (production-safe, non-blocking)
func ScanKeys(pattern string) ([]string, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}

	var keys []string
	var cursor uint64
	var err error

	for {
		var batch []string
		batch, cursor, err = Get().rdb.Scan(extCtx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan keys failed: %w", err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// DeletePattern deletes all keys matching pattern using SCAN (non-blocking)
func DeletePattern(pattern string) (int64, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}

	var totalDeleted int64
	var cursor uint64
	var err error

	for {
		var batch []string
		batch, cursor, err = Get().rdb.Scan(extCtx, cursor, pattern, 100).Result()
		if err != nil {
			return totalDeleted, fmt.Errorf("scan keys failed: %w", err)
		}

		if len(batch) > 0 {
			deleted, delErr := Get().rdb.Del(extCtx, batch...).Result()
			if delErr != nil {
				return totalDeleted, fmt.Errorf("delete keys failed: %w", delErr)
			}
			totalDeleted += deleted
		}

		if cursor == 0 {
			break
		}
	}

	return totalDeleted, nil
}

// Incr increments integer value at key
func Incr(key string) (int64, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}
	return Get().rdb.Incr(extCtx, key).Result()
}

// Decr decrements integer value at key
func Decr(key string) (int64, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}
	return Get().rdb.Decr(extCtx, key).Result()
}

// IncrBy increments integer value at key by delta
func IncrBy(key string, value int64) (int64, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}
	return Get().rdb.IncrBy(extCtx, key, value).Result()
}

// HSet sets hash field value (JSON-serialized)
func HSet(key string, field string, value interface{}) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal hash value failed: %w", err)
	}
	return Get().rdb.HSet(extCtx, key, field, data).Err()
}

// HGet gets hash field value (JSON-deserialized)
func HGet(key string, field string, dest interface{}) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	data, err := Get().rdb.HGet(extCtx, key, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("hget failed: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal hash value failed: %w", err)
	}

	return nil
}

// HGetAll gets all hash fields
func HGetAll(key string) (map[string]string, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.HGetAll(extCtx, key).Result()
}

// HDel deletes hash fields
func HDel(key string, fields ...string) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.HDel(extCtx, key, fields...).Err()
}

// ZAdd adds member to sorted set with score
func ZAdd(key string, score float64, member interface{}) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.ZAdd(extCtx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// ZRange gets sorted set members by rank (ascending)
func ZRange(key string, start, stop int64) ([]string, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.ZRange(extCtx, key, start, stop).Result()
}

// ZRevRange gets sorted set members by rank (descending)
func ZRevRange(key string, start, stop int64) ([]string, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.ZRevRange(extCtx, key, start, stop).Result()
}

// ZRangeWithScores gets sorted set members with scores (ascending)
func ZRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.ZRangeWithScores(extCtx, key, start, stop).Result()
}

// ZRevRangeWithScores gets sorted set members with scores (descending)
func ZRevRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.ZRevRangeWithScores(extCtx, key, start, stop).Result()
}

// ZRem removes members from sorted set
func ZRem(key string, members ...interface{}) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.ZRem(extCtx, key, members...).Err()
}

// ZCard gets sorted set cardinality
func ZCard(key string) (int64, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}
	return Get().rdb.ZCard(extCtx, key).Result()
}

// SAdd adds members to set
func SAdd(key string, members ...interface{}) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.SAdd(extCtx, key, members...).Err()
}

// SMembers gets all set members
func SMembers(key string) ([]string, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}
	return Get().rdb.SMembers(extCtx, key).Result()
}

// SIsMember checks if value is set member
func SIsMember(key string, member interface{}) (bool, error) {
	if !Available() {
		return false, fmt.Errorf("redis not available")
	}
	return Get().rdb.SIsMember(extCtx, key, member).Result()
}

// SRem removes members from set
func SRem(key string, members ...interface{}) error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.SRem(extCtx, key, members...).Err()
}

// FlushDB clears current database
func FlushDB() error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.FlushDB(extCtx).Err()
}

// FlushAll clears all databases and returns count of deleted keys
func FlushAll() (int64, error) {
	if !Available() {
		return 0, fmt.Errorf("redis not available")
	}

	// Get key count before flush
	keys, err := Get().rdb.Keys(extCtx, "*").Result()
	if err != nil {
		return 0, err
	}

	count := int64(len(keys))

	if err := Get().rdb.FlushDB(extCtx).Err(); err != nil {
		return 0, err
	}

	return count, nil
}

// HealthCheck pings Redis
func HealthCheck() error {
	if !Available() {
		return fmt.Errorf("redis not available")
	}
	return Get().rdb.Ping(extCtx).Err()
}

// CacheKey builds cache key from parts
func CacheKey(parts ...string) string {
	key := "newapi"
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// CacheWrapper wraps cache operations with key and TTL
type CacheWrapper struct {
	Key string
	TTL time.Duration
}

// GetOrSet gets from cache or executes function and caches result
func (c *CacheWrapper) GetOrSet(dest interface{}, fn func() (interface{}, error)) error {
	// Try cache first
	found, err := Get().GetJSON(c.Key, dest)
	if err != nil {
		log.Warn().Err(err).Str("key", c.Key).Msg("cache get failed, executing function")
	} else if found {
		return nil
	}

	// Execute function
	result, err := fn()
	if err != nil {
		return err
	}

	// Cache result
	if err := Get().Set(c.Key, result, c.TTL); err != nil {
		log.Warn().Err(err).Str("key", c.Key).Msg("cache set failed")
	}

	// Copy result to dest
	data, _ := json.Marshal(result)
	return json.Unmarshal(data, dest)
}

// Invalidate removes cache entry
func (c *CacheWrapper) Invalidate() error {
	return Get().Delete(c.Key)
}

// RedisInfo holds Redis statistics
type RedisInfo struct {
	KeyCount   int64
	MemoryUsed string
	HitRate    float64
	Uptime     string
}

// GetInfo gets Redis statistics
func GetInfo() (*RedisInfo, error) {
	if !Available() {
		return nil, fmt.Errorf("redis not available")
	}

	info := &RedisInfo{}

	// Get key count
	dbSize, err := Get().rdb.DBSize(extCtx).Result()
	if err == nil {
		info.KeyCount = dbSize
	}

	// Get memory usage
	memInfo, err := Get().rdb.Info(extCtx, "memory").Result()
	if err == nil {
		info.MemoryUsed = parseRedisInfoValue(memInfo, "used_memory_human")
	}

	// Get hit rate
	statsInfo, err := Get().rdb.Info(extCtx, "stats").Result()
	if err == nil {
		hits := parseRedisInfoInt(statsInfo, "keyspace_hits")
		misses := parseRedisInfoInt(statsInfo, "keyspace_misses")
		if hits+misses > 0 {
			info.HitRate = float64(hits) / float64(hits+misses) * 100
		}
	}

	// Get uptime
	serverInfo, err := Get().rdb.Info(extCtx, "server").Result()
	if err == nil {
		uptimeSecs := parseRedisInfoInt(serverInfo, "uptime_in_seconds")
		info.Uptime = formatUptime(uptimeSecs)
	}

	return info, nil
}

// parseRedisInfoValue extracts value from Redis INFO output
func parseRedisInfoValue(info, key string) string {
	lines := splitLines(info)
	for _, line := range lines {
		if len(line) > len(key)+1 && line[:len(key)] == key && line[len(key)] == ':' {
			return line[len(key)+1:]
		}
	}
	return "N/A"
}

// parseRedisInfoInt extracts integer value from Redis INFO output
func parseRedisInfoInt(info, key string) int64 {
	value := parseRedisInfoValue(info, key)
	if value == "N/A" {
		return 0
	}

	var result int64
	for _, c := range value {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		} else {
			break
		}
	}
	return result
}

// splitLines splits string by newlines
func splitLines(s string) []string {
	var lines []string
	var line []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, string(line))
			line = nil
		} else {
			line = append(line, s[i])
		}
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

// formatUptime formats uptime seconds to human-readable string
func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
