package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// MockMCPServer 模拟的 MCP Server
type MockMCPServer struct {
	port string
}

func NewMockMCPServer(port string) *MockMCPServer {
	return &MockMCPServer{port: port}
}

func (s *MockMCPServer) Start() {
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/tools", s.handleTools)
	http.HandleFunc("/execute", s.handleExecute)

	fmt.Printf("🚀 Mock MCP Server starting on http://localhost:%s\n", s.port)
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /health  - Health check")
	fmt.Println("  GET  /tools   - List available tools")
	fmt.Println("  POST /execute - Execute a tool")
	fmt.Println()

	if err := http.ListenAndServe(":"+s.port, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *MockMCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"server": "mock-mcp-server",
	})
}

func (s *MockMCPServer) handleTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tools := map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "weather",
				"description": "Get current weather information for a location",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]string{
							"type":        "string",
							"description": "City name or location",
						},
						"unit": map[string]string{
							"type":        "string",
							"description": "Temperature unit (celsius or fahrenheit)",
						},
					},
					"required": []string{"location"},
				},
			},
			{
				"name":        "translate",
				"description": "Translate text between languages",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]string{
							"type":        "string",
							"description": "Text to translate",
						},
						"from": map[string]string{
							"type":        "string",
							"description": "Source language code",
						},
						"to": map[string]string{
							"type":        "string",
							"description": "Target language code",
						},
					},
					"required": []string{"text", "to"},
				},
			},
			{
				"name":        "search",
				"description": "Search for information on the internet",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]string{
							"type":        "string",
							"description": "Search query",
						},
						"limit": map[string]string{
							"type":        "integer",
							"description": "Maximum number of results",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	json.NewEncoder(w).Encode(tools)
}

func (s *MockMCPServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Tool   string                 `json:"tool"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 模拟工具执行
	var result interface{}
	var success bool
	var errorMsg string

	switch request.Tool {
	case "weather":
		location, _ := request.Params["location"].(string)
		unit, ok := request.Params["unit"].(string)
		if !ok {
			unit = "celsius"
		}

		result = map[string]interface{}{
			"location":    location,
			"temperature": 22,
			"unit":        unit,
			"condition":   "Sunny",
			"humidity":    65,
		}
		success = true

	case "translate":
		text, _ := request.Params["text"].(string)
		to, _ := request.Params["to"].(string)

		// 简单的模拟翻译
		translations := map[string]string{
			"zh": "你好，世界",
			"es": "Hola Mundo",
			"fr": "Bonjour le monde",
			"ja": "こんにちは世界",
		}

		translated, ok := translations[to]
		if !ok {
			translated = text + " (translated to " + to + ")"
		}

		result = map[string]interface{}{
			"original":   text,
			"translated": translated,
			"from":       "en",
			"to":         to,
		}
		success = true

	case "search":
		query, _ := request.Params["query"].(string)
		limit := 5
		if l, ok := request.Params["limit"].(float64); ok {
			limit = int(l)
		}

		results := make([]map[string]interface{}, 0, limit)
		for i := 0; i < limit; i++ {
			results = append(results, map[string]interface{}{
				"title":   fmt.Sprintf("Result %d for: %s", i+1, query),
				"url":     fmt.Sprintf("https://example.com/result-%d", i+1),
				"snippet": fmt.Sprintf("This is a mock search result for query: %s", query),
			})
		}

		result = map[string]interface{}{
			"query":   query,
			"results": results,
			"total":   limit,
		}
		success = true

	default:
		success = false
		errorMsg = fmt.Sprintf("Unknown tool: %s", request.Tool)
	}

	response := map[string]interface{}{
		"success": success,
		"result":  result,
		"error":   errorMsg,
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	server := NewMockMCPServer("55987")
	server.Start()
}
