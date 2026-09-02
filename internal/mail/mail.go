package mail

import (
	"net/smtp"
	"strings"
)

type Sender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSender(host, port, username, password, from string) *Sender {
	return &Sender{host: host, port: port, username: username, password: password, from: from}
}

func (s *Sender) Enabled() bool {
	return s != nil && s.host != "" && s.from != ""
}

func (s *Sender) Send(to, subject, body string) error {
	if !s.Enabled() {
		return nil
	}
	message := strings.Join([]string{
		"From: Strata <" + s.from + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")
	var authentication smtp.Auth
	if s.username != "" {
		authentication = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	return smtp.SendMail(s.host+":"+s.port, authentication, s.from, []string{to}, []byte(message))
}
