package log

// Logger 统一日志接口
type Logger interface {
	// Debug 调试级别日志
	Debug(msg string, fields ...Field)

	// Info 信息级别日志
	Info(msg string, fields ...Field)

	// Warn 警告级别日志
	Warn(msg string, fields ...Field)

	// Error 错误级别日志
	Error(msg string, fields ...Field)

	// With 创建带有预设字段的子 Logger
	With(fields ...Field) Logger
}

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// String 创建字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建浮点数字段
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建布尔字段
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Any 创建任意类型字段
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Error 创建错误字段
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Duration 创建时长字段
func Duration(key string, value any) Field {
	return Field{Key: key, Value: value}
}
