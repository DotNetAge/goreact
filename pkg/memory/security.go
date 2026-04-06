package memory

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// SecurityReviewer provides security review for generated resources
type SecurityReviewer interface {
	ReviewSkill(ctx context.Context, skill *goreactcore.GeneratedSkillNode) (*ReviewResult, error)
	ReviewTool(ctx context.Context, tool *goreactcore.GeneratedToolNode) (*ReviewResult, error)
	ReviewMemoryItem(ctx context.Context, item *goreactcore.MemoryItemNode) (*ReviewResult, error)
	RunSecurityChecks(resource any, resourceType string) ([]SecurityCheckResult, error)
}

// ReviewResult represents the result of a security review
type ReviewResult struct {
	ResourceID   string               `json:"resource_id" yaml:"resource_id"`
	ResourceType string               `json:"resource_type" yaml:"resource_type"`
	Status       ReviewStatus         `json:"status" yaml:"status"`
	CheckResults []SecurityCheckResult `json:"check_results" yaml:"check_results"`
	ReviewedAt   time.Time            `json:"reviewed_at" yaml:"reviewed_at"`
	Reviewer     string               `json:"reviewer" yaml:"reviewer"`
	Comments     string               `json:"comments" yaml:"comments"`
}

// ReviewStatus represents the status of a review
type ReviewStatus string

const (
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusRejected ReviewStatus = "rejected"
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusWarning  ReviewStatus = "warning"
)

// SecurityCheckResult represents the result of a security check
type SecurityCheckResult struct {
	CheckName    string       `json:"check_name" yaml:"check_name"`
	CheckType    CheckType    `json:"check_type" yaml:"check_type"`
	Status       CheckStatus  `json:"status" yaml:"status"`
	Description  string       `json:"description" yaml:"description"`
	Details      string       `json:"details" yaml:"details"`
	Severity     Severity     `json:"severity" yaml:"severity"`
	Suggestions  []string     `json:"suggestions" yaml:"suggestions"`
}

// CheckType represents the type of security check
type CheckType string

const (
	CheckTypeSensitiveCommand CheckType = "sensitive_command"
	CheckTypePermissionScope   CheckType = "permission_scope"
	CheckTypeCodeInjection     CheckType = "code_injection"
	CheckTypeExternalDependency CheckType = "external_dependency"
	CheckTypeDuplicateDetection CheckType = "duplicate_detection"
	CheckTypeMaliciousPattern  CheckType = "malicious_pattern"
	CheckTypeDataExposure      CheckType = "data_exposure"
)

// CheckStatus represents the status of a check
type CheckStatus string

const (
	CheckStatusPass    CheckStatus = "pass"
	CheckStatusFail    CheckStatus = "fail"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusSkip    CheckStatus = "skip"
)

// Severity represents the severity level of a check result
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// SandboxExecutor executes code in a sandbox environment
type SandboxExecutor interface {
	ExecuteSkill(ctx context.Context, skill *goreactcore.GeneratedSkillNode, testData map[string]any) (*SandboxResult, error)
	ExecuteTool(ctx context.Context, tool *goreactcore.GeneratedToolNode, testData map[string]any) (*SandboxResult, error)
	IsAvailable() bool
}

// SandboxResult represents the result of a sandbox execution
type SandboxResult struct {
	Success      bool          `json:"success" yaml:"success"`
	Output       string        `json:"output" yaml:"output"`
	Error        string        `json:"error" yaml:"error"`
	Duration     time.Duration `json:"duration" yaml:"duration"`
	ResourceUsage ResourceUsage `json:"resource_usage" yaml:"resource_usage"`
}

// ResourceUsage represents resource usage during execution
type ResourceUsage struct {
	CPUTime    time.Duration `json:"cpu_time" yaml:"cpu_time"`
	MemoryUsed int64         `json:"memory_used" yaml:"memory_used"`
	DiskUsed   int64         `json:"disk_used" yaml:"disk_used"`
}

// securityReviewer implements SecurityReviewer
type securityReviewer struct {
	config         *SecurityConfig
	dangerousPatterns []*regexp.Regexp
	sensitiveCommands []string
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	EnableAutoReview     bool         `json:"enable_auto_review" yaml:"enable_auto_review"`
	EnableSandbox        bool         `json:"enable_sandbox" yaml:"enable_sandbox"`
	ReviewPolicy         goreactcommon.ReviewPolicy `json:"review_policy" yaml:"review_policy"`
	MaxExecutionTime     time.Duration `json:"max_execution_time" yaml:"max_execution_time"`
	MaxMemoryUsage       int64         `json:"max_memory_usage" yaml:"max_memory_usage"`
	BlockedCommands      []string      `json:"blocked_commands" yaml:"blocked_commands"`
	AllowedDependencies  []string      `json:"allowed_dependencies" yaml:"allowed_dependencies"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableAutoReview: true,
		EnableSandbox:    true,
		ReviewPolicy:     goreactcommon.ReviewHybrid,
		MaxExecutionTime: 30 * time.Second,
		MaxMemoryUsage:   100 * 1024 * 1024, // 100MB
		BlockedCommands: []string{
			"rm -rf",
			"DROP TABLE",
			"DELETE FROM",
			"TRUNCATE",
			"format",
			"mkfs",
			"dd if=",
			":(){ :|:& };:",
			"chmod 777",
			"chown root",
		},
		AllowedDependencies: []string{
			"fmt",
			"strings",
			"strconv",
			"time",
			"encoding/json",
			"encoding/xml",
		},
	}
}

// NewSecurityReviewer creates a new SecurityReviewer
func NewSecurityReviewer(config *SecurityConfig) SecurityReviewer {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	
	reviewer := &securityReviewer{
		config:         config,
		sensitiveCommands: config.BlockedCommands,
		dangerousPatterns: compileDangerousPatterns(),
	}
	
	return reviewer
}

// compileDangerousPatterns compiles dangerous code patterns
func compileDangerousPatterns() []*regexp.Regexp {
	patterns := []string{
		`rm\s+-rf\s+/`,
		`DROP\s+TABLE`,
		`DELETE\s+FROM\s+\w+\s*;`,
		`eval\s*\(`,
		`exec\s*\(`,
		`system\s*\(`,
		`subprocess\.call`,
		`os\.system`,
		`__import__`,
		`importlib\.import_module`,
		`\$\{.*\}`,
		`<%.*%>`,
		`\{\{.*\}\}`,
	}
	
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			compiled = append(compiled, re)
		}
	}
	
	return compiled
}

// ReviewSkill reviews a generated skill
func (r *securityReviewer) ReviewSkill(ctx context.Context, skill *goreactcore.GeneratedSkillNode) (*ReviewResult, error) {
	result := &ReviewResult{
		ResourceID:   skill.Name,
		ResourceType: "skill",
		Status:       ReviewStatusPending,
		ReviewedAt:   time.Now(),
		CheckResults: []SecurityCheckResult{},
	}
	
	// Run security checks
	checkResults, err := r.RunSecurityChecks(skill, "skill")
	if err != nil {
		return nil, err
	}
	result.CheckResults = checkResults
	
	// Determine overall status
	result.Status = r.determineStatus(checkResults)
	
	return result, nil
}

// ReviewTool reviews a generated tool
func (r *securityReviewer) ReviewTool(ctx context.Context, tool *goreactcore.GeneratedToolNode) (*ReviewResult, error) {
	result := &ReviewResult{
		ResourceID:   tool.Name,
		ResourceType: "tool",
		Status:       ReviewStatusPending,
		ReviewedAt:   time.Now(),
		CheckResults: []SecurityCheckResult{},
	}
	
	// Run security checks
	checkResults, err := r.RunSecurityChecks(tool, "tool")
	if err != nil {
		return nil, err
	}
	result.CheckResults = checkResults
	
	// Determine overall status
	result.Status = r.determineStatus(checkResults)
	
	return result, nil
}

// ReviewMemoryItem reviews a generated memory item
func (r *securityReviewer) ReviewMemoryItem(ctx context.Context, item *goreactcore.MemoryItemNode) (*ReviewResult, error) {
	result := &ReviewResult{
		ResourceID:   item.Name,
		ResourceType: "memory_item",
		Status:       ReviewStatusPending,
		ReviewedAt:   time.Now(),
		CheckResults: []SecurityCheckResult{},
	}
	
	// Run security checks
	checkResults, err := r.RunSecurityChecks(item, "memory_item")
	if err != nil {
		return nil, err
	}
	result.CheckResults = checkResults
	
	// Determine overall status
	result.Status = r.determineStatus(checkResults)
	
	return result, nil
}

// RunSecurityChecks runs all security checks on a resource
func (r *securityReviewer) RunSecurityChecks(resource any, resourceType string) ([]SecurityCheckResult, error) {
	results := []SecurityCheckResult{}
	
	// Check 1: Sensitive command detection
	results = append(results, r.checkSensitiveCommands(resource, resourceType))
	
	// Check 2: Code injection detection
	results = append(results, r.checkCodeInjection(resource, resourceType))
	
	// Check 3: Malicious pattern detection
	results = append(results, r.checkMaliciousPatterns(resource, resourceType))
	
	// Check 4: Data exposure check
	results = append(results, r.checkDataExposure(resource, resourceType))
	
	// Check 5: Permission scope check (for skill/tool)
	if resourceType == "skill" || resourceType == "tool" {
		results = append(results, r.checkPermissionScope(resource, resourceType))
	}
	
	return results, nil
}

// checkSensitiveCommands checks for sensitive commands
func (r *securityReviewer) checkSensitiveCommands(resource any, resourceType string) SecurityCheckResult {
	result := SecurityCheckResult{
		CheckName:   "Sensitive Command Detection",
		CheckType:   CheckTypeSensitiveCommand,
		Status:      CheckStatusPass,
		Description: "Checks for dangerous system commands",
		Severity:    SeverityCritical,
		Suggestions: []string{},
	}
	
	var content string
	switch v := resource.(type) {
	case *goreactcore.GeneratedSkillNode:
		content = v.Content
	case *goreactcore.GeneratedToolNode:
		content = v.Code
	case *goreactcore.MemoryItemNode:
		content = v.Content
	}
	
	contentLower := strings.ToLower(content)
	for _, cmd := range r.sensitiveCommands {
		if strings.Contains(contentLower, strings.ToLower(cmd)) {
			result.Status = CheckStatusFail
			result.Details = fmt.Sprintf("Found sensitive command: %s", cmd)
			result.Suggestions = append(result.Suggestions, fmt.Sprintf("Remove or modify the command: %s", cmd))
		}
	}
	
	return result
}

// checkCodeInjection checks for code injection vulnerabilities
func (r *securityReviewer) checkCodeInjection(resource any, resourceType string) SecurityCheckResult {
	result := SecurityCheckResult{
		CheckName:   "Code Injection Detection",
		CheckType:   CheckTypeCodeInjection,
		Status:      CheckStatusPass,
		Description: "Checks for potential code injection vulnerabilities",
		Severity:    SeverityHigh,
		Suggestions: []string{},
	}
	
	var content string
	switch v := resource.(type) {
	case *goreactcore.GeneratedSkillNode:
		content = v.Template
	case *goreactcore.GeneratedToolNode:
		content = v.Code
	}
	
	// Check for template injection patterns
	injectionPatterns := []string{
		"${", "{{", "<%", "%>", 
	}
	
	for _, pattern := range injectionPatterns {
		if strings.Contains(content, pattern) {
			result.Status = CheckStatusWarning
			result.Details = fmt.Sprintf("Found potential injection pattern: %s", pattern)
			result.Suggestions = append(result.Suggestions, "Ensure proper input sanitization")
		}
	}
	
	return result
}

// checkMaliciousPatterns checks for malicious code patterns
func (r *securityReviewer) checkMaliciousPatterns(resource any, resourceType string) SecurityCheckResult {
	result := SecurityCheckResult{
		CheckName:   "Malicious Pattern Detection",
		CheckType:   CheckTypeMaliciousPattern,
		Status:      CheckStatusPass,
		Description: "Checks for known malicious code patterns",
		Severity:    SeverityCritical,
		Suggestions: []string{},
	}
	
	var content string
	switch v := resource.(type) {
	case *goreactcore.GeneratedSkillNode:
		content = v.Content
	case *goreactcore.GeneratedToolNode:
		content = v.Code
	}
	
	for _, pattern := range r.dangerousPatterns {
		if pattern.MatchString(content) {
			result.Status = CheckStatusFail
			result.Details = fmt.Sprintf("Found malicious pattern: %s", pattern.String())
			result.Suggestions = append(result.Suggestions, "Remove the malicious code pattern")
		}
	}
	
	return result
}

// checkDataExposure checks for potential data exposure
func (r *securityReviewer) checkDataExposure(resource any, resourceType string) SecurityCheckResult {
	result := SecurityCheckResult{
		CheckName:   "Data Exposure Check",
		CheckType:   CheckTypeDataExposure,
		Status:      CheckStatusPass,
		Description: "Checks for potential data exposure risks",
		Severity:    SeverityHigh,
		Suggestions: []string{},
	}
	
	var content string
	switch v := resource.(type) {
	case *goreactcore.GeneratedSkillNode:
		content = v.Content
	case *goreactcore.GeneratedToolNode:
		content = v.Code
	case *goreactcore.MemoryItemNode:
		content = v.Content
	}
	
	// Check for sensitive data patterns
	sensitivePatterns := []string{
		"password",
		"secret",
		"api_key",
		"token",
		"credential",
	}
	
	contentLower := strings.ToLower(content)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(contentLower, pattern) {
			result.Status = CheckStatusWarning
			result.Details = fmt.Sprintf("Found potential sensitive data reference: %s", pattern)
			result.Suggestions = append(result.Suggestions, "Ensure sensitive data is properly handled")
		}
	}
	
	return result
}

// checkPermissionScope checks permission scope
func (r *securityReviewer) checkPermissionScope(resource any, resourceType string) SecurityCheckResult {
	result := SecurityCheckResult{
		CheckName:   "Permission Scope Check",
		CheckType:   CheckTypePermissionScope,
		Status:      CheckStatusPass,
		Description: "Checks if permissions are within acceptable scope",
		Severity:    SeverityMedium,
		Suggestions: []string{},
	}
	
	switch v := resource.(type) {
	case *goreactcore.GeneratedSkillNode:
		// Check allowed tools
		if len(v.AllowedTools) > 10 {
			result.Status = CheckStatusWarning
			result.Details = "Too many allowed tools may increase security risk"
			result.Suggestions = append(result.Suggestions, "Consider reducing the number of allowed tools")
		}
	case *goreactcore.GeneratedToolNode:
		// Check security level
		if v.SecurityLevel == goreactcommon.LevelHighRisk {
			result.Status = CheckStatusWarning
			result.Details = "Tool is marked as high risk"
			result.Suggestions = append(result.Suggestions, "Ensure high-risk operations are properly controlled")
		}
	}
	
	return result
}

// determineStatus determines overall review status
func (r *securityReviewer) determineStatus(results []SecurityCheckResult) ReviewStatus {
	hasFail := false
	hasWarning := false
	
	for _, result := range results {
		if result.Status == CheckStatusFail {
			if result.Severity == SeverityCritical || result.Severity == SeverityHigh {
				return ReviewStatusRejected
			}
			hasFail = true
		} else if result.Status == CheckStatusWarning {
			hasWarning = true
		}
	}
	
	if hasFail {
		return ReviewStatusRejected
	}
	if hasWarning {
		return ReviewStatusWarning
	}
	return ReviewStatusApproved
}

// sandboxExecutor implements SandboxExecutor
type sandboxExecutor struct {
	config *SecurityConfig
}

// NewSandboxExecutor creates a new SandboxExecutor
func NewSandboxExecutor(config *SecurityConfig) SandboxExecutor {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	return &sandboxExecutor{config: config}
}

// ExecuteSkill executes a skill in sandbox
func (e *sandboxExecutor) ExecuteSkill(ctx context.Context, skill *goreactcore.GeneratedSkillNode, testData map[string]any) (*SandboxResult, error) {
	start := time.Now()
	
	result := &SandboxResult{
		Success:  true,
		Duration: time.Since(start),
		ResourceUsage: ResourceUsage{
			CPUTime:    0,
			MemoryUsed: 0,
			DiskUsed:   0,
		},
	}
	
	// Simulate sandbox execution
	// In production, this would execute in an isolated environment
	
	return result, nil
}

// ExecuteTool executes a tool in sandbox
func (e *sandboxExecutor) ExecuteTool(ctx context.Context, tool *goreactcore.GeneratedToolNode, testData map[string]any) (*SandboxResult, error) {
	start := time.Now()
	
	result := &SandboxResult{
		Success:  true,
		Duration: time.Since(start),
		ResourceUsage: ResourceUsage{
			CPUTime:    0,
			MemoryUsed: 0,
			DiskUsed:   0,
		},
	}
	
	// Simulate sandbox execution
	// In production, this would execute in an isolated environment
	
	return result, nil
}

// IsAvailable checks if sandbox is available
func (e *sandboxExecutor) IsAvailable() bool {
	return e.config.EnableSandbox
}

// ReviewService provides review service for generated resources
type ReviewService interface {
	SubmitForReview(ctx context.Context, resource any, resourceType string) (*ReviewResult, error)
	GetReviewStatus(ctx context.Context, resourceID string) (*ReviewResult, error)
	ApproveResource(ctx context.Context, resourceID string, reviewer string) error
	RejectResource(ctx context.Context, resourceID string, reviewer string, reason string) error
	ListPendingReviews(ctx context.Context) ([]*ReviewResult, error)
}

// reviewService implements ReviewService
type reviewService struct {
	reviewer SecurityReviewer
	sandbox  SandboxExecutor
	memory   *Memory
	results  map[string]*ReviewResult
}

// NewReviewService creates a new ReviewService
func NewReviewService(memory *Memory, config *SecurityConfig) ReviewService {
	return &reviewService{
		reviewer: NewSecurityReviewer(config),
		sandbox:  NewSandboxExecutor(config),
		memory:   memory,
		results:  make(map[string]*ReviewResult),
	}
}

// SubmitForReview submits a resource for review
func (s *reviewService) SubmitForReview(ctx context.Context, resource any, resourceType string) (*ReviewResult, error) {
	var result *ReviewResult
	var err error
	
	switch resourceType {
	case "skill":
		skill, ok := resource.(*goreactcore.GeneratedSkillNode)
		if !ok {
			return nil, fmt.Errorf("invalid skill type")
		}
		result, err = s.reviewer.ReviewSkill(ctx, skill)
	case "tool":
		tool, ok := resource.(*goreactcore.GeneratedToolNode)
		if !ok {
			return nil, fmt.Errorf("invalid tool type")
		}
		result, err = s.reviewer.ReviewTool(ctx, tool)
	case "memory_item":
		item, ok := resource.(*goreactcore.MemoryItemNode)
		if !ok {
			return nil, fmt.Errorf("invalid memory item type")
		}
		result, err = s.reviewer.ReviewMemoryItem(ctx, item)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	
	if err != nil {
		return nil, err
	}
	
	s.results[result.ResourceID] = result
	
	return result, nil
}

// GetReviewStatus gets the review status of a resource
func (s *reviewService) GetReviewStatus(ctx context.Context, resourceID string) (*ReviewResult, error) {
	result, ok := s.results[resourceID]
	if !ok {
		return nil, fmt.Errorf("review result not found: %s", resourceID)
	}
	return result, nil
}

// ApproveResource approves a resource
func (s *reviewService) ApproveResource(ctx context.Context, resourceID string, reviewer string) error {
	result, ok := s.results[resourceID]
	if !ok {
		return fmt.Errorf("review result not found: %s", resourceID)
	}
	
	result.Status = ReviewStatusApproved
	result.Reviewer = reviewer
	result.ReviewedAt = time.Now()
	
	return nil
}

// RejectResource rejects a resource
func (s *reviewService) RejectResource(ctx context.Context, resourceID string, reviewer string, reason string) error {
	result, ok := s.results[resourceID]
	if !ok {
		return fmt.Errorf("review result not found: %s", resourceID)
	}
	
	result.Status = ReviewStatusRejected
	result.Reviewer = reviewer
	result.Comments = reason
	result.ReviewedAt = time.Now()
	
	return nil
}

// ListPendingReviews lists all pending reviews
func (s *reviewService) ListPendingReviews(ctx context.Context) ([]*ReviewResult, error) {
	pending := make([]*ReviewResult, 0)
	for _, result := range s.results {
		if result.Status == ReviewStatusPending {
			pending = append(pending, result)
		}
	}
	return pending, nil
}
