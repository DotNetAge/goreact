package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// Cron cron 表达式工具，支持解析和计划任务管理
type Cron struct {
	info *core.ToolInfo
	// accessor provides access to the reactor's scheduler (set via SetAccessor).
	accessor ReactorAccessor
}

// NewCronTool 创建 cron 工具
func NewCronTool() core.FuncTool {
	return &Cron{
		info: &core.ToolInfo{
			Name:          "cron",
			Description:   "Cron expression tool. Operations: 'parse'|'next'|'validate' for expression handling; 'schedule'|'unschedule'|'list_schedules'|'enable_schedule'|'disable_schedule' for scheduled task management. Params: {operation: string, expression: 'cron_expr', from: 'time_string', count: number, name: 'task_name', prompt: 'task_prompt', schedule_id: 'id'}",
			SecurityLevel: core.LevelSafe,
		},
	}
}

// SetAccessor sets the reactor accessor for scheduler access.
func (c *Cron) SetAccessor(a ReactorAccessor) {
	c.accessor = a
}

func (c *Cron) Info() *core.ToolInfo {
	return c.info
}

// Execute 执行 cron 操作
func (c *Cron) Execute(ctx context.Context, params map[string]any) (any, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "validate":
		// 验证 cron 表达式
		expression, ok := params["expression"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'expression' parameter")
		}
		return c.validate(expression)

	case "parse":
		// 解析 cron 表达式
		expression, ok := params["expression"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'expression' parameter")
		}
		return c.parse(expression)

	case "next":
		// 计算下一个或多个执行时间
		expression, ok := params["expression"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'expression' parameter")
		}
		fromStr, ok := params["from"].(string)
		if !ok {
			// 如果没有指定 from，使用当前时间
			fromStr = time.Now().Format(time.RFC3339)
		}
		count := 1
		if cnt, ok := params["count"].(float64); ok {
			count = int(cnt)
		}
		return c.next(expression, fromStr, count)

	case "schedule":
		// 注册定时任务
		return c.schedule(params)

	case "unschedule":
		// 删除定时任务
		return c.unschedule(params)

	case "list_schedules":
		// 列出所有定时任务
		return c.listSchedules()

	case "enable_schedule":
		// 启用定时任务
		return c.enableSchedule(params)

	case "disable_schedule":
		// 禁用定时任务
		return c.disableSchedule(params)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// validate 验证 cron 表达式
func (c *Cron) validate(expression string) (map[string]any, error) {
	fields, err := parseCronExpression(expression)
	if err != nil {
		return map[string]any{"valid": false, "error": err.Error()}, nil
	}

	return map[string]any{
		"valid":  true,
		"fields": fields,
	}, nil
}

// parse 解析 cron 表达式
func (c *Cron) parse(expression string) (map[string]any, error) {
	fields, err := parseCronExpression(expression)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"expression": expression,
		"minute":     fieldToString(fields[0]),
		"hour":       fieldToString(fields[1]),
		"day":        fieldToString(fields[2]),
		"month":      fieldToString(fields[3]),
		"weekday":    fieldToString(fields[4]),
	}, nil
}

// next 计算下一个或多个执行时间
func (c *Cron) next(expression, fromStr string, count int) ([]string, error) {
	fields, err := parseCronExpression(expression)
	if err != nil {
		return nil, err
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'from' time: %w", err)
	}

	var results []string
	current := from.Add(time.Minute) // 从下一分钟开始查找

	// 限制最大搜索次数，防止无限循环
	maxIterations := 366 * 24 * 60 // 一年的分钟数
	iterations := 0

	for len(results) < count && iterations < maxIterations {
		if c.matches(current, fields) {
			results = append(results, current.Format(time.RFC3339))
		}
		current = current.Add(time.Minute)
		iterations++
	}

	if len(results) < count {
		return nil, fmt.Errorf("could not find %d occurrences within one year", count)
	}

	return results, nil
}

// matches 检查时间是否匹配 cron 表达式
func (c *Cron) matches(t time.Time, fields [][]int) bool {
	minute := t.Minute()
	hour := t.Hour()
	day := t.Day()
	month := int(t.Month())
	weekday := int(t.Weekday()) // Sunday = 0

	return containsInt(fields[0], minute) &&
		containsInt(fields[1], hour) &&
		containsInt(fields[2], day) &&
		containsInt(fields[3], month) &&
		containsInt(fields[4], weekday)
}

// containsInt 检查 int 切片是否包含某个值
func containsInt(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// parseCronExpression 解析 cron 表达式为字段
func parseCronExpression(expr string) ([][]int, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("invalid cron expression: expected 5 fields, got %d", len(fields))
	}

	result := make([][]int, 5)

	// 分钟 (0-59)
	var err error
	result[0], err = parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}

	// 小时 (0-23)
	result[1], err = parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}

	// 日期 (1-31)
	result[2], err = parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day field: %w", err)
	}

	// 月份 (1-12)
	result[3], err = parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}

	// 星期 (0-6, 0=Sunday)
	result[4], err = parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid weekday field: %w", err)
	}

	return result, nil
}

// parseField 解析单个 cron 字段
func parseField(field string, min, max int) ([]int, error) {
	var values []int

	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "*" {
			// 通配符：所有值
			for i := min; i <= max; i++ {
				values = append(values, i)
			}
			continue
		}

		if strings.Contains(part, "/") {
			// 步长：*/n 或 start-end/n
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return nil, fmt.Errorf("invalid step format: %s", part)
			}

			step, err := strconv.Atoi(stepParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid step value: %s", stepParts[1])
			}
			if step <= 0 {
				return nil, fmt.Errorf("step must be positive: %d", step)
			}

			start, end := min, max
			if stepParts[0] != "*" {
				if strings.Contains(stepParts[0], "-") {
					rangeParts := strings.Split(stepParts[0], "-")
					if len(rangeParts) != 2 {
						return nil, fmt.Errorf("invalid range format: %s", stepParts[0])
					}
					start, err = strconv.Atoi(rangeParts[0])
					if err != nil {
						return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
					}
					end, err = strconv.Atoi(rangeParts[1])
					if err != nil {
						return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
					}
				} else {
					start, err = strconv.Atoi(stepParts[0])
					if err != nil {
						return nil, fmt.Errorf("invalid start value: %s", stepParts[0])
					}
				}
			}

			if start < min || start > max {
				return nil, fmt.Errorf("start value %d out of range [%d-%d]", start, min, max)
			}
			if end < min || end > max {
				return nil, fmt.Errorf("end value %d out of range [%d-%d]", end, min, max)
			}

			for i := start; i <= end; i += step {
				values = append(values, i)
			}
			continue
		}

		if strings.Contains(part, "-") {
			// 范围：start-end
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
			}

			if start < min || start > max {
				return nil, fmt.Errorf("range start %d out of range [%d-%d]", start, min, max)
			}
			if end < min || end > max {
				return nil, fmt.Errorf("range end %d out of range [%d-%d]", end, min, max)
			}
			if start > end {
				return nil, fmt.Errorf("range start %d is greater than end %d", start, end)
			}

			for i := start; i <= end; i++ {
				values = append(values, i)
			}
			continue
		}

		// 单个值
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", part)
		}

		if val < min || val > max {
			return nil, fmt.Errorf("value %d out of range [%d-%d]", val, min, max)
		}

		values = append(values, val)
	}

	return values, nil
}

// fieldToString 将字段转换为字符串表示
func fieldToString(field []int) string {
	if len(field) == 0 {
		return ""
	}

	// 如果是所有值，返回 "*"
	min := field[0]
	max := field[len(field)-1]

	// 检查是否覆盖了完整范围（从最小值到最大值连续）
	fullRange := max - min + 1
	if len(field) == fullRange && fullRange > 1 {
		allMatch := true
		for i, v := range field {
			if v != min+i {
				allMatch = false
				break
			}
		}
		if allMatch {
			// 如果覆盖了完整范围，返回 "start-end" 格式
			return fmt.Sprintf("%d-%d", min, max)
		}
	}

	// 否则返回逗号分隔的列表
	strs := make([]string, len(field))
	for i, v := range field {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

// --- Scheduled task management operations ---

// schedule registers a new scheduled task via the reactor's CronScheduler.
func (c *Cron) schedule(params map[string]any) (map[string]any, error) {
	if c.accessor == nil {
		return nil, fmt.Errorf("scheduler not available: accessor not configured")
	}

	scheduler := c.accessor.Scheduler()
	if scheduler == nil {
		return nil, fmt.Errorf("scheduler not configured. Use reactor.WithScheduler() to enable scheduled tasks")
	}

	name, _ := params["name"].(string)
	expression, ok := params["expression"].(string)
	if !ok || expression == "" {
		return nil, fmt.Errorf("missing or invalid 'expression' parameter")
	}
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("missing or invalid 'prompt' parameter")
	}

	if name == "" {
		name = fmt.Sprintf("schedule_%d", time.Now().Unix())
	}

	id, err := scheduler.Schedule(name, expression, prompt)
	if err != nil {
		return nil, err
	}

	task := scheduler.Get(id)
	return map[string]any{
		"id":         id,
		"name":       name,
		"expression": expression,
		"prompt":     prompt,
		"next_run":   task.NextRunAt.Format(time.RFC3339),
		"enabled":    true,
	}, nil
}

// unschedule removes a scheduled task by ID.
func (c *Cron) unschedule(params map[string]any) (map[string]any, error) {
	if c.accessor == nil {
		return nil, fmt.Errorf("scheduler not available: accessor not configured")
	}

	scheduler := c.accessor.Scheduler()
	if scheduler == nil {
		return nil, fmt.Errorf("scheduler not configured")
	}

	id, ok := params["schedule_id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("missing or invalid 'schedule_id' parameter")
	}

	removed := scheduler.Unschedule(id)
	if !removed {
		return map[string]any{"removed": false, "id": id}, nil
	}

	return map[string]any{"removed": true, "id": id}, nil
}

// listSchedules returns all scheduled tasks.
func (c *Cron) listSchedules() (map[string]any, error) {
	if c.accessor == nil {
		return nil, fmt.Errorf("scheduler not available: accessor not configured")
	}

	scheduler := c.accessor.Scheduler()
	if scheduler == nil {
		return map[string]any{"tasks": []any{}, "count": 0}, nil
	}

	tasks := scheduler.List()
	items := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		item := map[string]any{
			"id":        t.ID,
			"name":      t.Name,
			"expression": t.Expression,
			"prompt":    t.Prompt,
			"enabled":   t.Enabled,
			"run_count": t.RunCount,
		}
		if !t.NextRunAt.IsZero() {
			item["next_run"] = t.NextRunAt.Format(time.RFC3339)
		}
		if !t.LastRunAt.IsZero() {
			item["last_run"] = t.LastRunAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	return map[string]any{
		"tasks": items,
		"count": len(items),
	}, nil
}

// enableSchedule enables a scheduled task.
func (c *Cron) enableSchedule(params map[string]any) (map[string]any, error) {
	if c.accessor == nil {
		return nil, fmt.Errorf("scheduler not available: accessor not configured")
	}

	scheduler := c.accessor.Scheduler()
	if scheduler == nil {
		return nil, fmt.Errorf("scheduler not configured")
	}

	id, ok := params["schedule_id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("missing or invalid 'schedule_id' parameter")
	}

	if err := scheduler.Enable(id); err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "enabled": true}, nil
}

// disableSchedule disables a scheduled task.
func (c *Cron) disableSchedule(params map[string]any) (map[string]any, error) {
	if c.accessor == nil {
		return nil, fmt.Errorf("scheduler not available: accessor not configured")
	}

	scheduler := c.accessor.Scheduler()
	if scheduler == nil {
		return nil, fmt.Errorf("scheduler not configured")
	}

	id, ok := params["schedule_id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("missing or invalid 'schedule_id' parameter")
	}

	if err := scheduler.Disable(id); err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "enabled": false}, nil
}
