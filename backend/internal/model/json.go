package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSON 是一个存储 JSON 文本的自定义类型，映射到 SQLite 的 TEXT 列。
// 实现 driver.Valuer / sql.Scanner，可直接 json.Marshal / json.Unmarshal。
type JSON []byte

// Value 实现 driver.Valuer。
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner。
func (j *JSON) Scan(v interface{}) error {
	if v == nil {
		*j = nil
		return nil
	}
	switch b := v.(type) {
	case []byte:
		*j = append((*j)[:0], b...)
	case string:
		*j = []byte(b)
	default:
		return fmt.Errorf("model.JSON: 不支持的类型 %T", v)
	}
	return nil
}

// MarshalJSON 透传内部 JSON 文本。
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 原样保存。
func (j *JSON) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}

// Set 用任意可序列化对象填充 JSON。
func (j *JSON) Set(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*j = b
	return nil
}

// Unmarshal 把 JSON 反序列化到目标对象。
func (j JSON) Unmarshal(v interface{}) error {
	if len(j) == 0 {
		return nil
	}
	return json.Unmarshal(j, v)
}
