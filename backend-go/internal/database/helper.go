package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Manager 提供与 backend 兼容的数据库操作接口
// 这是一个适配层，将 GORM 操作包装成类似 sqlx 的 API
type Manager struct {
	DB   *gorm.DB
	IsPG bool
}

// GetManager 返回主数据库的 Manager 实例
func GetManager() *Manager {
	isPG := GetDBEngine() == "postgres"
	return &Manager{
		DB:   GetMainDB(),
		IsPG: isPG,
	}
}

// Query 执行查询并返回 map 结果集
func (m *Manager) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	rows, err := m.DB.Raw(query, args...).Rows()
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
	result := m.DB.Exec(query, args...)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// RebindQuery 重新绑定查询占位符（GORM 自动处理，这里保持兼容）
func (m *Manager) RebindQuery(query string) string {
	if m.IsPG {
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
	var exists bool
	var query string

	if m.IsPG {
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

	err := m.DB.Raw(query, tableName).Scan(&exists).Error
	return exists, err
}

// ColumnExists 检查列是否存在
func (m *Manager) ColumnExists(tableName, columnName string) bool {
	var exists bool
	var query string

	if m.IsPG {
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

	err := m.DB.Raw(query, tableName, columnName).Scan(&exists).Error
	return err == nil && exists
}

// Placeholder 返回数据库占位符（GORM 自动处理，保持兼容）
func (m *Manager) Placeholder(index int) string {
	if m.IsPG {
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
