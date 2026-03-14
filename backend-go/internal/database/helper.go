package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Manager 提供与 backend 兼容的数据库操作接口
// 这是一个适配层，将 GORM 操作包装成类似 sqlx 的 API
//
// 通过 GetManager() 创建的实例使用惰性获取：DB 连接在首次使用时动态获取，
// 因此可安全地在包级变量初始化阶段调用（此时 mainDB 可能尚未初始化）。
type Manager struct {
	DB   *gorm.DB
	IsPG bool
	lazy bool // 惰性模式：DB/IsPG 在使用时动态获取
}

// GetManager 返回主数据库的惰性 Manager 实例
// 可安全地在包级变量初始化中调用，DB 连接在首次使用时动态获取
func GetManager() *Manager {
	return &Manager{lazy: true}
}

// getDB 返回底层 *gorm.DB，惰性模式下动态获取最新的 mainDB
func (m *Manager) getDB() *gorm.DB {
	if m.lazy {
		return GetMainDB()
	}
	return m.DB
}

// isPostgres 返回是否为 PostgreSQL，惰性模式下动态获取
func (m *Manager) isPostgres() bool {
	if m.lazy {
		return GetDBEngine() == "postgres"
	}
	return m.IsPG
}

// Query 执行查询并返回 map 结果集
func (m *Manager) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	db := m.getDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}
	var results []map[string]interface{}
	rows, err := db.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 构建 map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 转换 []byte 为 string
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// QueryOne 执行查询并返回单行结果
func (m *Manager) QueryOne(query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := m.Query(query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Execute 执行非查询语句（INSERT, UPDATE, DELETE）
func (m *Manager) Execute(query string, args ...interface{}) (int64, error) {
	db := m.getDB()
	if db == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}
	result := db.Exec(query, args...)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// RebindQuery 重新绑定查询占位符（GORM 自动处理，这里保持兼容）
func (m *Manager) RebindQuery(query string) string {
	if m.isPostgres() {
		// 将 ? 转换为 $1, $2, ...
		count := 0
		var result strings.Builder
		for _, ch := range query {
			if ch == '?' {
				count++
				result.WriteString(fmt.Sprintf("$%d", count))
			} else {
				result.WriteRune(ch)
			}
		}
		return result.String()
	}
	return query
}

// TableExists 检查表是否存在
func (m *Manager) TableExists(tableName string) (bool, error) {
	db := m.getDB()
	if db == nil {
		return false, fmt.Errorf("数据库连接未初始化")
	}
	var exists bool
	var query string

	if m.isPostgres() {
		query = `SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = ?
		)`
	} else {
		// MySQL
		query = `SELECT COUNT(*) > 0 FROM information_schema.tables
			WHERE table_schema = DATABASE()
			AND table_name = ?`
	}

	err := db.Raw(query, tableName).Scan(&exists).Error
	return exists, err
}

// ColumnExists 检查列是否存在
func (m *Manager) ColumnExists(tableName, columnName string) bool {
	db := m.getDB()
	if db == nil {
		return false
	}
	var exists bool
	var query string

	if m.isPostgres() {
		query = `SELECT EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = ?
			AND column_name = ?
		)`
	} else {
		// MySQL
		query = `SELECT COUNT(*) > 0 FROM information_schema.columns
			WHERE table_schema = DATABASE()
			AND table_name = ?
			AND column_name = ?`
	}

	err := db.Raw(query, tableName, columnName).Scan(&exists).Error
	return err == nil && exists
}

// Placeholder 返回数据库占位符（GORM 自动处理，保持兼容）
func (m *Manager) Placeholder(index int) string {
	if m.isPostgres() {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

// QueryWithTimeout 执行带超时的查询（兼容 backend API）
// backend 的签名是: QueryWithTimeout(timeout time.Duration, query string, args ...interface{})
func (m *Manager) QueryWithTimeout(timeout interface{}, query string, args ...interface{}) ([]map[string]interface{}, error) {
	// 简化实现：直接调用 Query，GORM 会使用默认超时
	// timeout 参数被忽略，如果需要严格的超时控制，可以使用 context.WithTimeout
	return m.Query(query, args...)
}
