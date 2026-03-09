package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ray/goreact/pkg/tool"
)

// PropertyType 参数类型
type PropertyType string

const (
	String PropertyType = "string"
	Number PropertyType = "number"
	Int    PropertyType = "integer"
	Bool   PropertyType = "boolean"
	Array  PropertyType = "array"
	Object PropertyType = "object"
)

// ParamDef 参数定义
type ParamDef struct {
	name        string
	typ         PropertyType
	description string
	required    bool
	defaultVal  any
	enum        []string
	min         float64
	max         float64
	hasRange    bool
}

// Param 创建参数定义
func Param(name string, typ PropertyType, description string) *ParamDef {
	return &ParamDef{
		name:        name,
		typ:         typ,
		description: description,
	}
}

// Required 标记为必需参数
func (p *ParamDef) Required() *ParamDef {
	p.required = true
	return p
}

// Default 设置默认值
func (p *ParamDef) Default(val any) *ParamDef {
	p.defaultVal = val
	return p
}

// Enum 设置枚举值
func (p *ParamDef) Enum(values ...string) *ParamDef {
	p.enum = values
	return p
}

// Range 设置数值范围
func (p *ParamDef) Range(min, max float64) *ParamDef {
	p.min = min
	p.max = max
	p.hasRange = true
	return p
}

// Schema 参数 Schema
type Schema struct {
	params []*ParamDef
}

// Define 定义 Schema
func Define(params ...*ParamDef) *Schema {
	return &Schema{params: params}
}

// ValidatedParams 已验证的参数
type ValidatedParams struct {
	values map[string]any
}

// GetString 获取字符串参数
func (p ValidatedParams) GetString(key string) string {
	if v, ok := p.values[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// GetInt 获取整数参数
func (p ValidatedParams) GetInt(key string) int {
	if v, ok := p.values[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	return 0
}

// GetFloat64 获取浮点数参数
func (p ValidatedParams) GetFloat64(key string) float64 {
	if v, ok := p.values[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case string:
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

// GetBool 获取布尔参数
func (p ValidatedParams) GetBool(key string) bool {
	if v, ok := p.values[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return val == "true" || val == "1" || val == "yes"
		case int:
			return val != 0
		case float64:
			return val != 0
		}
	}
	return false
}

// Has 检查参数是否存在
func (p ValidatedParams) Has(key string) bool {
	_, ok := p.values[key]
	return ok
}

// Validate 验证参数
func (s *Schema) Validate(params map[string]any) (ValidatedParams, error) {
	result := make(map[string]any)

	for _, param := range s.params {
		val, exists := params[param.name]

		// 检查必需参数
		if !exists {
			if param.required {
				return ValidatedParams{}, fmt.Errorf("parameter '%s' is required", param.name)
			}
			// 使用默认值
			if param.defaultVal != nil {
				result[param.name] = param.defaultVal
			}
			continue
		}

		// 类型转换和验证
		converted, err := s.convertAndValidate(param, val)
		if err != nil {
			return ValidatedParams{}, fmt.Errorf("parameter '%s': %w", param.name, err)
		}

		result[param.name] = converted
	}

	return ValidatedParams{values: result}, nil
}

// convertAndValidate 转换和验证参数值
func (s *Schema) convertAndValidate(param *ParamDef, val any) (any, error) {
	// 类型转换
	converted, err := s.convertType(param.typ, val)
	if err != nil {
		return nil, err
	}

	// Enum 验证
	if len(param.enum) > 0 {
		strVal := fmt.Sprintf("%v", converted)
		valid := false
		for _, e := range param.enum {
			if strVal == e {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("must be one of: %s, got '%v'", strings.Join(param.enum, ", "), converted)
		}
	}

	// 范围验证
	if param.hasRange && (param.typ == Number || param.typ == Int) {
		var numVal float64
		switch v := converted.(type) {
		case float64:
			numVal = v
		case int:
			numVal = float64(v)
		case int64:
			numVal = float64(v)
		}

		if numVal < param.min || numVal > param.max {
			return nil, fmt.Errorf("must be between %.0f and %.0f, got %.0f", param.min, param.max, numVal)
		}
	}

	return converted, nil
}

// convertType 类型转换
func (s *Schema) convertType(typ PropertyType, val any) (any, error) {
	switch typ {
	case String:
		return fmt.Sprintf("%v", val), nil

	case Number:
		switch v := val.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to number", v)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to number", val)
		}

	case Int:
		switch v := val.(type) {
		case int:
			return v, nil
		case int64:
			return int(v), nil
		case float64:
			return int(v), nil
		case string:
			i, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to integer", v)
			}
			return i, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to integer", val)
		}

	case Bool:
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			return v == "true" || v == "1" || v == "yes", nil
		case int:
			return v != 0, nil
		case float64:
			return v != 0, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to boolean", val)
		}

	default:
		return val, nil
	}
}

// Tool Schema-based Tool
type Tool struct {
	name        string
	description string
	schema      *Schema
	handler     func(ValidatedParams) (any, error)
}

// NewTool 创建 Schema-based Tool
func NewTool(name, description string, schema *Schema, handler func(ValidatedParams) (any, error)) *Tool {
	return &Tool{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *Tool) Description() string {
	return t.description
}

// Execute 执行工具
func (t *Tool) Execute(params map[string]any) (any, error) {
	// 验证参数
	validated, err := t.schema.Validate(params)
	if err != nil {
		return nil, err
	}

	// 调用处理器
	return t.handler(validated)
}

// SchemaJSON 返回 JSON Schema
func (t *Tool) SchemaJSON() string {
	schema := map[string]any{
		"name":        t.name,
		"description": t.description,
		"parameters": map[string]any{
			"type":       "object",
			"properties": t.buildProperties(),
			"required":   t.buildRequired(),
		},
	}

	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}

func (t *Tool) buildProperties() map[string]any {
	props := make(map[string]any)
	for _, param := range t.schema.params {
		prop := map[string]any{
			"type":        string(param.typ),
			"description": param.description,
		}
		if len(param.enum) > 0 {
			prop["enum"] = param.enum
		}
		if param.defaultVal != nil {
			prop["default"] = param.defaultVal
		}
		if param.hasRange {
			prop["minimum"] = param.min
			prop["maximum"] = param.max
		}
		props[param.name] = prop
	}
	return props
}

func (t *Tool) buildRequired() []string {
	var required []string
	for _, param := range t.schema.params {
		if param.required {
			required = append(required, param.name)
		}
	}
	return required
}

// 确保 Tool 实现了 tool.Tool 接口
var _ tool.Tool = (*Tool)(nil)

// UserError 用户错误（不应该重试）
type UserError struct {
	message string
}

// NewUserError 创建用户错误
func NewUserError(format string, args ...any) error {
	return &UserError{
		message: fmt.Sprintf(format, args...),
	}
}

func (e *UserError) Error() string {
	return e.message
}

// IsUserError 判断是否是用户错误
func IsUserError(err error) bool {
	_, ok := err.(*UserError)
	return ok
}
