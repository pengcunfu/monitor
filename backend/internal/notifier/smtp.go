package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"monitor/internal/model"
)

type smtpConfig struct {
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	User               string   `json:"user"`
	Password           string   `json:"password"`
	From               string   `json:"from"`
	To                 []string `json:"to"`
	TLS                bool     `json:"tls"`                 // true=465 SSL 直连；false=587 STARTTLS
	InsecureSkipVerify bool     `json:"insecure_skip_verify"` // 自签名证书时置 true
}

type smtpNotifier struct {
	cfg smtpConfig
}

func newSMTP(c *model.NotificationChannel) (Notifier, error) {
	var cfg smtpConfig
	if err := c.ConfigJSON.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("SMTP 配置解析失败: %w", err)
	}
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" || len(cfg.To) == 0 {
		return nil, fmt.Errorf("SMTP 配置不完整（host/port/from/to 必填）")
	}
	return &smtpNotifier{cfg: cfg}, nil
}

func (n *smtpNotifier) Send(ctx context.Context, msg *Message) error {
	addr := fmt.Sprintf("%s:%d", n.cfg.Host, n.cfg.Port)
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		n.cfg.From, n.cfg.From, msg.Title, msg.Content)
	auth := smtp.PlainAuth("", n.cfg.User, n.cfg.Password, n.cfg.Host)
	to := n.cfg.To

	if n.cfg.TLS {
		return n.sendSSL(ctx, addr, auth, to, []byte(body))
	}
	return smtp.SendMail(addr, auth, n.cfg.From, to, []byte(body))
}

// sendSSL 使用隐式 TLS（465）直连发送。
func (n *smtpNotifier) sendSSL(ctx context.Context, addr string, auth smtp.Auth, to []string, body []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: n.cfg.InsecureSkipVerify,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = tlsConn.Close()
		return err
	}
	c, err := smtp.NewClient(tlsConn, host)
	if err != nil {
		_ = tlsConn.Close()
		return err
	}
	if err := c.Auth(auth); err != nil {
		_ = c.Close()
		return err
	}
	if err := c.Mail(n.cfg.From); err != nil {
		_ = c.Close()
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			_ = c.Close()
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		_ = c.Close()
		return err
	}
	if _, err := w.Write(body); err != nil {
		_ = c.Close()
		return err
	}
	if err := w.Close(); err != nil {
		_ = c.Close()
		return err
	}
	return c.Quit()
}
