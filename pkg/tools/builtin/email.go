package builtin

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/DotNetAge/goreact/pkg/tools"
	"io"
	"net/smtp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// Email 邮件工具
type Email struct {
	config EmailConfig
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTP SMTPConfig
	IMAP IMAPConfig
}

// SMTPConfig SMTP 配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

// IMAPConfig IMAP 配置
type IMAPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
}

// NewEmail 创建邮件工具
func NewEmail(config EmailConfig) *Email {
	return &Email{config: config}
}

// Name 返回工具名称
func (e *Email) Name() string {
	return "email"
}

// Description 返回工具描述
func (e *Email) Description() string {
	return "Email operations: send, send_html, list, read, search, delete, move, mark_read, mark_unread"
}

// Execute 执行邮件操作
// SecurityLevel returns the tool's security risk level
func (t *Email) SecurityLevel() tools.SecurityLevel {
	return tools.LevelHighRisk // Default, needs manual update for risky tools
}

func (e *Email) Execute(ctx context.Context, params map[string]any) (any, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "send":
		return e.send(params)
	case "send_html":
		return e.sendHTML(params)
	case "list":
		return e.list(params)
	case "read":
		return e.read(params)
	case "search":
		return e.search(params)
	case "delete":
		return e.delete(params)
	case "move":
		return e.move(params)
	case "mark_read":
		return e.markAsRead(params)
	case "mark_unread":
		return e.markAsUnread(params)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// send 发送文本邮件
func (e *Email) send(params map[string]any) (any, error) {
	to, ok := params["to"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'to' parameter")
	}

	subject, ok := params["subject"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'subject' parameter")
	}

	body, ok := params["body"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'body' parameter")
	}

	// 构建邮件
	msg := e.buildMessage(to, subject, body, params, false)

	// 发送邮件
	err := e.sendMail(to, msg)
	if err != nil {
		return nil, e.formatError("send", err)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Email sent to %s", to),
	}, nil
}

// sendHTML 发送 HTML 邮件
func (e *Email) sendHTML(params map[string]any) (any, error) {
	to, ok := params["to"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'to' parameter")
	}

	subject, ok := params["subject"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'subject' parameter")
	}

	html, ok := params["html"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'html' parameter")
	}

	// 构建邮件
	msg := e.buildMessage(to, subject, html, params, true)

	// 发送邮件
	err := e.sendMail(to, msg)
	if err != nil {
		return nil, e.formatError("send_html", err)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("HTML email sent to %s", to),
	}, nil
}

// list 列出邮件
func (e *Email) list(params map[string]any) (any, error) {
	// 连接 IMAP 服务器
	c, err := e.connectIMAP()
	if err != nil {
		return nil, e.formatError("list", err)
	}
	defer c.Logout()

	// 选择文件夹
	folder := "INBOX"
	if f, ok := params["folder"].(string); ok && f != "" {
		folder = f
	}

	mbox, err := c.Select(folder, false)
	if err != nil {
		return nil, e.formatError("list", err)
	}

	// 获取邮件数量
	limit := 20
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > uint32(limit) {
		from = mbox.Messages - uint32(limit) + 1
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags}, messages)
	}()

	emails := []map[string]any{}
	for msg := range messages {
		email := map[string]any{
			"id":      msg.SeqNum,
			"subject": msg.Envelope.Subject,
			"date":    msg.Envelope.Date.Format(time.RFC3339),
			"unread":  !contains(msg.Flags, imap.SeenFlag),
		}

		if len(msg.Envelope.From) > 0 {
			email["from"] = msg.Envelope.From[0].Address()
		}

		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, e.formatError("list", err)
	}

	return map[string]any{
		"success": true,
		"emails":  emails,
		"total":   len(emails),
	}, nil
}

// read 读取邮件
func (e *Email) read(params map[string]any) (any, error) {
	messageID, ok := params["message_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message_id' parameter")
	}

	// 连接 IMAP 服务器
	c, err := e.connectIMAP()
	if err != nil {
		return nil, e.formatError("read", err)
	}
	defer c.Logout()

	// 选择文件夹
	folder := "INBOX"
	if f, ok := params["folder"].(string); ok && f != "" {
		folder = f
	}

	_, err = c.Select(folder, false)
	if err != nil {
		return nil, e.formatError("read", err)
	}

	// 获取邮件
	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(messageID))

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody + "[]"}, messages)
	}()

	msg := <-messages
	if msg == nil {
		return nil, fmt.Errorf("message not found")
	}

	if err := <-done; err != nil {
		return nil, e.formatError("read", err)
	}

	// 解析邮件正文
	var section imap.BodySectionName
	r := msg.GetBody(&section)
	if r == nil {
		return nil, fmt.Errorf("failed to get message body")
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, e.formatError("read", err)
	}

	body := ""
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			b, _ := io.ReadAll(p.Body)
			body += string(b)
		case *mail.AttachmentHeader:
			// 跳过附件
			_ = h
		}
	}

	email := map[string]any{
		"id":      messageID,
		"subject": msg.Envelope.Subject,
		"body":    body,
		"date":    msg.Envelope.Date.Format(time.RFC3339),
	}

	if len(msg.Envelope.From) > 0 {
		email["from"] = msg.Envelope.From[0].Address()
	}

	// 标记为已读
	if markRead, ok := params["mark_as_read"].(bool); !ok || markRead {
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []any{imap.SeenFlag}
		c.Store(seqset, item, flags, nil)
	}

	return map[string]any{
		"success": true,
		"email":   email,
	}, nil
}

// search 搜索邮件
func (e *Email) search(params map[string]any) (any, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'query' parameter")
	}

	// 连接 IMAP 服务器
	c, err := e.connectIMAP()
	if err != nil {
		return nil, e.formatError("search", err)
	}
	defer c.Logout()

	// 选择文件夹
	folder := "INBOX"
	if f, ok := params["folder"].(string); ok && f != "" {
		folder = f
	}

	_, err = c.Select(folder, false)
	if err != nil {
		return nil, e.formatError("search", err)
	}

	// 构建搜索条件
	criteria := imap.NewSearchCriteria()
	criteria.Text = []string{query}

	// 按发件人过滤
	if from, ok := params["from"].(string); ok && from != "" {
		criteria.Header.Add("From", from)
	}

	// 按主题过滤
	if subject, ok := params["subject"].(string); ok && subject != "" {
		criteria.Header.Add("Subject", subject)
	}

	// 搜索
	ids, err := c.Search(criteria)
	if err != nil {
		return nil, e.formatError("search", err)
	}

	return map[string]any{
		"success": true,
		"count":   len(ids),
		"ids":     ids,
	}, nil
}

// delete 删除邮件
func (e *Email) delete(params map[string]any) (any, error) {
	messageID, ok := params["message_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message_id' parameter")
	}

	// 连接 IMAP 服务器
	c, err := e.connectIMAP()
	if err != nil {
		return nil, e.formatError("delete", err)
	}
	defer c.Logout()

	// 选择文件夹
	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, e.formatError("delete", err)
	}

	// 标记为删除
	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(messageID))

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []any{imap.DeletedFlag}
	err = c.Store(seqset, item, flags, nil)
	if err != nil {
		return nil, e.formatError("delete", err)
	}

	// 永久删除
	err = c.Expunge(nil)
	if err != nil {
		return nil, e.formatError("delete", err)
	}

	return map[string]any{
		"success": true,
		"message": "Email deleted",
	}, nil
}

// move 移动邮件
func (e *Email) move(params map[string]any) (any, error) {
	messageID, ok := params["message_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message_id' parameter")
	}

	folder, ok := params["folder"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'folder' parameter")
	}

	// 连接 IMAP 服务器
	c, err := e.connectIMAP()
	if err != nil {
		return nil, e.formatError("move", err)
	}
	defer c.Logout()

	// 选择源文件夹
	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, e.formatError("move", err)
	}

	// 移动邮件
	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(messageID))

	err = c.Move(seqset, folder)
	if err != nil {
		return nil, e.formatError("move", err)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Email moved to %s", folder),
	}, nil
}

// markAsRead 标记为已读
func (e *Email) markAsRead(params map[string]any) (any, error) {
	return e.setFlag(params, imap.SeenFlag, true)
}

// markAsUnread 标记为未读
func (e *Email) markAsUnread(params map[string]any) (any, error) {
	return e.setFlag(params, imap.SeenFlag, false)
}

// setFlag 设置标志
func (e *Email) setFlag(params map[string]any, flag string, add bool) (any, error) {
	messageID, ok := params["message_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message_id' parameter")
	}

	// 连接 IMAP 服务器
	c, err := e.connectIMAP()
	if err != nil {
		return nil, e.formatError("set_flag", err)
	}
	defer c.Logout()

	// 选择文件夹
	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, e.formatError("set_flag", err)
	}

	// 设置标志
	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(messageID))

	var item imap.StoreItem
	if add {
		item = imap.FormatFlagsOp(imap.AddFlags, true)
	} else {
		item = imap.FormatFlagsOp(imap.RemoveFlags, true)
	}

	flags := []any{flag}
	err = c.Store(seqset, item, flags, nil)
	if err != nil {
		return nil, e.formatError("set_flag", err)
	}

	return map[string]any{
		"success": true,
		"message": "Flag updated",
	}, nil
}

// buildMessage 构建邮件消息
func (e *Email) buildMessage(to, subject, body string, params map[string]any, isHTML bool) string {
	from := e.config.SMTP.From
	if from == "" {
		from = e.config.SMTP.Username
	}

	msg := fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", to)

	// CC
	if cc, ok := params["cc"].([]any); ok && len(cc) > 0 {
		ccList := []string{}
		for _, c := range cc {
			if addr, ok := c.(string); ok {
				ccList = append(ccList, addr)
			}
		}
		if len(ccList) > 0 {
			msg += fmt.Sprintf("Cc: %s\r\n", strings.Join(ccList, ", "))
		}
	}

	msg += fmt.Sprintf("Subject: %s\r\n", subject)

	if isHTML {
		msg += "MIME-Version: 1.0\r\n"
		msg += "Content-Type: text/html; charset=UTF-8\r\n"
	}

	msg += "\r\n" + body

	return msg
}

// sendMail 发送邮件
func (e *Email) sendMail(to, msg string) error {
	auth := smtp.PlainAuth("", e.config.SMTP.Username, e.config.SMTP.Password, e.config.SMTP.Host)

	addr := fmt.Sprintf("%s:%d", e.config.SMTP.Host, e.config.SMTP.Port)

	if e.config.SMTP.TLS {
		// 使用 TLS
		tlsConfig := &tls.Config{
			ServerName: e.config.SMTP.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, e.config.SMTP.Host)
		if err != nil {
			return err
		}
		defer c.Close()

		if err = c.Auth(auth); err != nil {
			return err
		}

		if err = c.Mail(e.config.SMTP.From); err != nil {
			return err
		}

		if err = c.Rcpt(to); err != nil {
			return err
		}

		w, err := c.Data()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		return c.Quit()
	}

	// 不使用 TLS
	return smtp.SendMail(addr, auth, e.config.SMTP.From, []string{to}, []byte(msg))
}

// connectIMAP 连接 IMAP 服务器
func (e *Email) connectIMAP() (*client.Client, error) {
	addr := fmt.Sprintf("%s:%d", e.config.IMAP.Host, e.config.IMAP.Port)

	var c *client.Client
	var err error

	if e.config.IMAP.TLS {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}

	if err != nil {
		return nil, err
	}

	// 登录
	err = c.Login(e.config.IMAP.Username, e.config.IMAP.Password)
	if err != nil {
		c.Logout()
		return nil, err
	}

	return c, nil
}

// formatError 格式化错误消息
func (e *Email) formatError(operation string, err error) error {
	msg := fmt.Sprintf("Email %s failed: %v", operation, err)

	// 提供友好的建议
	suggestions := e.getSuggestions(err.Error())
	if suggestions != "" {
		msg += "\n\nSuggestions:\n" + suggestions
	}

	return fmt.Errorf("%s", msg)
}

// getSuggestions 根据错误提供建议
func (e *Email) getSuggestions(errMsg string) string {
	errMsg = strings.ToLower(errMsg)

	if strings.Contains(errMsg, "authentication failed") || strings.Contains(errMsg, "invalid credentials") {
		return "1. Check username and password\n" +
			"2. Enable 'Less secure app access' (Gmail)\n" +
			"3. Use app-specific password\n" +
			"4. Check if 2FA is enabled"
	}

	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "dial tcp") {
		return "1. Check SMTP/IMAP server address\n" +
			"2. Verify port number\n" +
			"3. Check firewall settings\n" +
			"4. Verify TLS/SSL settings"
	}

	if strings.Contains(errMsg, "timeout") {
		return "1. Check network connection\n" +
			"2. Try again later\n" +
			"3. Check if server is accessible"
	}

	if strings.Contains(errMsg, "mailbox does not exist") {
		return "1. Check folder name\n" +
			"2. List available folders\n" +
			"3. Use 'INBOX' for inbox"
	}

	return ""
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
