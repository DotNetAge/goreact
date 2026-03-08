package ollama

// GenerateRequest Ollama 生成请求
type GenerateRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Stream      bool                   `json:"stream"`
	Temperature float64                `json:"temperature,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// GenerateResponse Ollama 生成响应
type GenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`
}

// ErrorResponse Ollama 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}
