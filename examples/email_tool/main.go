package main

import (
	"fmt"

	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== Email Tool 示例 ===")

	// 配置邮件工具
	config := builtin.EmailConfig{
		SMTP: builtin.SMTPConfig{
			Host:     "smtp.gmail.com",
			Port:     587,
			Username: "your-email@gmail.com",
			Password: "your-app-password",
			From:     "your-email@gmail.com",
			TLS:      true,
		},
		IMAP: builtin.IMAPConfig{
			Host:     "imap.gmail.com",
			Port:     993,
			Username: "your-email@gmail.com",
			Password: "your-app-password",
			TLS:      true,
		},
	}

	emailTool := builtin.NewEmail(config)

	// ============================================================
	// 1. Send - 发送文本邮件
	// ============================================================
	fmt.Println("\n--- 1. Send Text Email ---")

	result, err := emailTool.Execute(map[string]any{
		"operation": "send",
		"to":        "recipient@example.com",
		"subject":   "Test Email from GoReAct",
		"body":      "This is a test email sent from GoReAct email tool.",
		"cc":        []any{"cc@example.com"},
	})

	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email sent: %v\n", result)
	}

	// ============================================================
	// 2. SendHTML - 发送 HTML 邮件
	// ============================================================
	fmt.Println("\n--- 2. Send HTML Email ---")

	html := `
		<html>
		<body>
			<h1>Hello from GoReAct!</h1>
			<p>This is an <strong>HTML</strong> email.</p>
			<ul>
				<li>Feature 1</li>
				<li>Feature 2</li>
				<li>Feature 3</li>
			</ul>
		</body>
		</html>
	`

	result, err = emailTool.Execute(map[string]any{
		"operation": "send_html",
		"to":        "recipient@example.com",
		"subject":   "HTML Email from GoReAct",
		"html":      html,
	})

	if err != nil {
		fmt.Printf("❌ SendHTML failed: %v\n", err)
	} else {
		fmt.Printf("✅ HTML email sent: %v\n", result)
	}

	// ============================================================
	// 3. List - 列出邮件
	// ============================================================
	fmt.Println("\n--- 3. List Emails ---")

	result, err = emailTool.Execute(map[string]any{
		"operation": "list",
		"folder":    "INBOX",
		"limit":     10,
	})

	if err != nil {
		fmt.Printf("❌ List failed: %v\n", err)
	} else {
		fmt.Printf("✅ Emails listed: %v\n", result)
	}

	// ============================================================
	// 4. Read - 读取邮件
	// ============================================================
	fmt.Println("\n--- 4. Read Email ---")

	result, err = emailTool.Execute(map[string]any{
		"operation":    "read",
		"message_id":   1,
		"mark_as_read": true,
	})

	if err != nil {
		fmt.Printf("❌ Read failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email read: %v\n", result)
	}

	// ============================================================
	// 5. Search - 搜索邮件
	// ============================================================
	fmt.Println("\n--- 5. Search Emails ---")

	result, err = emailTool.Execute(map[string]any{
		"operation": "search",
		"query":     "meeting",
		"from":      "boss@example.com",
	})

	if err != nil {
		fmt.Printf("❌ Search failed: %v\n", err)
	} else {
		fmt.Printf("✅ Search results: %v\n", result)
	}

	// ============================================================
	// 6. MarkAsRead - 标记为已读
	// ============================================================
	fmt.Println("\n--- 6. Mark As Read ---")

	result, err = emailTool.Execute(map[string]any{
		"operation":  "mark_read",
		"message_id": 1,
	})

	if err != nil {
		fmt.Printf("❌ MarkAsRead failed: %v\n", err)
	} else {
		fmt.Printf("✅ Marked as read: %v\n", result)
	}

	// ============================================================
	// 7. Move - 移动邮件
	// ============================================================
	fmt.Println("\n--- 7. Move Email ---")

	result, err = emailTool.Execute(map[string]any{
		"operation":  "move",
		"message_id": 1,
		"folder":     "Archive",
	})

	if err != nil {
		fmt.Printf("❌ Move failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email moved: %v\n", result)
	}

	// ============================================================
	// 8. Delete - 删除邮件
	// ============================================================
	fmt.Println("\n--- 8. Delete Email ---")

	result, err = emailTool.Execute(map[string]any{
		"operation":  "delete",
		"message_id": 1,
	})

	if err != nil {
		fmt.Printf("❌ Delete failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email deleted: %v\n", result)
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("\n注意：请替换配置中的邮箱地址和密码后再运行此示例")
}
