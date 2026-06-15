package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/pkg/logger"
)

type AlertLevel string

const (
	LevelInfo    AlertLevel = "INFO"
	LevelWarning AlertLevel = "WARNING"
	LevelCritical AlertLevel = "CRITICAL"
)

type Alert struct {
	Title      string
	Message    string
	Level      AlertLevel
	Timestamp  time.Time
	Source     string
}

type Notifier struct {
	telegramToken  string
	telegramChat   string
	discordWebhook string
	smtpServer     string
	smtpPort       int
	smtpUser       string
	smtpPass       string
	emailFrom      string
	emailTo        []string
	enabled        bool
}

func New(telegramToken, telegramChat, discordWebhook, smtpServer string, smtpPort int, smtpUser, smtpPass, emailFrom string, emailTo []string) *Notifier {
	return &Notifier{
		telegramToken:  telegramToken,
		telegramChat:   telegramChat,
		discordWebhook: discordWebhook,
		smtpServer:     smtpServer,
		smtpPort:       smtpPort,
		smtpUser:       smtpUser,
		smtpPass:       smtpPass,
		emailFrom:      emailFrom,
		emailTo:        emailTo,
		enabled:        telegramToken != "" || discordWebhook != "" || smtpServer != "",
	}
}

func (n *Notifier) Send(ctx context.Context, alert Alert) error {
	if !n.enabled {
		logger.Debug("alerting disabled, skipping alert: %s", alert.Title)
		return nil
	}

	logger.Info("sending alert [%s]: %s", alert.Level, alert.Title)

	var lastErr error

	if n.telegramToken != "" && n.telegramChat != "" {
		if err := n.sendTelegram(ctx, alert); err != nil {
			lastErr = err
			logger.Error("telegram alert failed: %v", err)
		}
	}

	if n.discordWebhook != "" {
		if err := n.sendDiscord(ctx, alert); err != nil {
			lastErr = err
			logger.Error("discord alert failed: %v", err)
		}
	}

	if n.smtpServer != "" && len(n.emailTo) > 0 {
		if err := n.sendEmail(ctx, alert); err != nil {
			lastErr = err
			logger.Error("email alert failed: %v", err)
		}
	}

	return lastErr
}

func (n *Notifier) sendTelegram(ctx context.Context, alert Alert) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.telegramToken)
	text := fmt.Sprintf("*[%s] %s*\n%s", alert.Level, alert.Title, alert.Message)

	body := map[string]string{
		"chat_id":    n.telegramChat,
		"text":       text,
		"parse_mode": "Markdown",
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API returned %d", resp.StatusCode)
	}
	return nil
}

func (n *Notifier) sendDiscord(ctx context.Context, alert Alert) error {
	color := 0xFFFF00
	switch alert.Level {
	case LevelCritical:
		color = 0xFF0000
	case LevelInfo:
		color = 0x00FF00
	}

	embed := map[string]interface{}{
		"title":       fmt.Sprintf("[%s] %s", alert.Level, alert.Title),
		"description": alert.Message,
		"color":       color,
		"timestamp":   alert.Timestamp.Format(time.RFC3339),
		"footer": map[string]string{
			"text": "NetCheck Alert",
		},
	}

	body := map[string]interface{}{
		"embeds": []interface{}{embed},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", n.discordWebhook, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (n *Notifier) sendEmail(ctx context.Context, alert Alert) error {
	auth := smtp.PlainAuth("", n.smtpUser, n.smtpPass, n.smtpServer)

	subject := fmt.Sprintf("[NetCheck] %s - %s", alert.Level, alert.Title)
	body := fmt.Sprintf("Level: %s\r\nTimestamp: %s\r\nSource: %s\r\n\r\n%s",
		alert.Level, alert.Timestamp.Format(time.RFC3339), alert.Source, alert.Message)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		n.emailFrom, strings.Join(n.emailTo, ","), subject, body)

	addr := fmt.Sprintf("%s:%d", n.smtpServer, n.smtpPort)

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, n.emailFrom, n.emailTo, []byte(msg))
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func Enabled(n *Notifier) bool {
	return n != nil && n.enabled
}
