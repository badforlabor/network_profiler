/**
 * Auth :   liubo
 * Date :   2020/2/8 17:59
 * Comment: 发送邮件
 */

package main

import (
	"gopkg.in/gomail.v2"
)

type IEmail interface {
	SendEmailTo(to, subject, body string) error
}
func NewEmail(host string, port int, account string, password string) IEmail {
	var e = &oneEmail{}
	e.Host = host
	e.Port = port
	e.Account = account
	e.Password = password

	return e
}
func NewEmailExample() IEmail {
	return NewEmail("smtp.qq.com", 465, "563568850@qq.com", "password-xxxxxx")
}

type oneEmail struct {
	Host     string
	Port     int
	Account  string
	Password string
}
func (self *oneEmail) SendEmail(subject, body string) error {
	var to = "505700330@qq.com"
	return self.SendEmailTo(to, subject, body)
}
func (self *oneEmail) SendEmailTo(to, subject, body string) error {

	var from = self.Account

	m := gomail.NewMessage()
	m.SetAddressHeader("From", from, from)  // 发件人
	m.SetHeader("To",  // 收件人
		m.FormatAddress(to, to),
	)
	m.SetHeader("Subject", subject)  // 主题
	m.SetBody("text/html", body)  // 正文
	//m.Attach("/home/Alex/lolcat.jpg")

	//d := gomail.NewDialer("smtp.qq.com", 465, from, "xaodkjrskzzmbfec")
	d := gomail.NewDialer(self.Host, self.Port, self.Account, self.Password)
	var err = d.DialAndSend(m)

	return err
}

var emailWorker = NewEmail("smtp.qq.com", 465, "563568850@qq.com", "----")
func sendEmail(subject, body string) bool {
	var to = "505700330@qq.com"
	var e = emailWorker.SendEmailTo(to, subject, body)
	if e != nil {
		netLog.Warnln("发送邮件错误:", e.Error())
	}
	return e == nil
}
