// Simple smtp client to send email(support Chinese).
// Work well with qq, 163, mac mail app, google mail.
package main

// https://github.com/WillCastor/smtpx

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	VerStr       string = "2023-07-07.07"
	DefServerStr string = "smtp.163.com:465"

	MaxLineLength int = 76
)

var boundary = "xxxxxxxx"

func init() {
	boundary = generateBoundary()
}

type Attachment struct {
	Filename string
	Data     []byte
}

type Sender struct {
	Host     string
	Port     int
	Address  string
	Password string
	Name     string
}

type Letter struct {
	address     string
	name        string
	Subject     string
	content     string
	receivers   []string
	carbonCopy  []string
	attachments []Attachment
	body        []byte
	mutex       sync.Mutex
}

func NewSender(host string, port int, name, address, password string) *Sender {
	return &Sender{Host: host, Port: port, Address: address, Password: password, Name: name}
}

func (s *Sender) NewLetter() *Letter {
	return &Letter{address: s.Address, name: s.Name}
}

func NewAttachment(filename string, data []byte) Attachment {
	return Attachment{Filename: filename, Data: data}
}

func (l *Letter) AddReceivers(addresses ...string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, address := range addresses {
		l.receivers = append(l.receivers, address)
	}
}

func (l *Letter) AddCarbonCopy(addresses ...string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, address := range addresses {
		l.carbonCopy = append(l.carbonCopy, address)
	}
}

func (l *Letter) SetSubject(subject string) {
	l.Subject = subject
}

func (l *Letter) SetContent(content string) {
	l.content = content
}

func (l *Letter) AddAttachments(attachments ...Attachment) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, item := range attachments {
		l.attachments = append(l.attachments, item)
	}
}

func (l *Letter) AddAttachment(filename string, data []byte) {
	l.AddAttachments(Attachment{Filename: filename, Data: data})
}

func (s *Sender) Send(l *Letter) error {
	l.build()
	auth := smtp.PlainAuth(
		"",
		s.Address,
		s.Password,
		s.Host,
	)
	return s.sendMailUsingTLS(
		fmt.Sprintf("%s:%d", s.Host, s.Port),
		auth,
		s.Address,
		l.receivers,
		l.carbonCopy,
		l.body)
}

func (l *Letter) Dump() {
	if len(l.body) == 0 {
		l.build()
	}
	fmt.Println(string(l.body))
}

func (l *Letter) build() {
	l.body = []byte{}
	l.buildHeader()
	l.buildContent()
	l.buildAttachments()
	l.buildEnd()
}

func (s *Sender) dial(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		log.Println("Dialing Error:", err)
		return nil, err
	}
	host, _, _ := net.SplitHostPort(addr)
	return smtp.NewClient(conn, host)
}

func (s *Sender) sendMailUsingTLS(addr string, auth smtp.Auth, from string,
	to []string, cc []string, msg []byte) (err error) {
	client, err := s.dial(addr)
	if err != nil {
		log.Println("Create smpt client error:", err)
		return err
	}
	defer client.Close()
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				log.Println("Error during AUTH", err)
				return err
			}
		}
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return err
		}
	}
	for _, addr := range cc {
		if err = client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return client.Quit()
}

func (l *Letter) buildHeader() {
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(
		fmt.Sprintf("From: %s<%s>\nTo: %s\nCC: %s\nSubject: %s\nMIME-Version: 1.0\n",
			l.name, l.address, strings.Join(l.receivers, ","), strings.Join(l.carbonCopy, ","), l.Subject))
	buffer.WriteString(
		fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n\n", boundary))
	buffer.WriteString(fmt.Sprintf("--%s\n", boundary))
	l.body = buffer.Bytes()
}

func (l *Letter) buildContent() {
	buffer := bytes.NewBuffer(l.body)
	buffer.WriteString("Content-Type: text/plain; charset=UTF-8\n")
	buffer.WriteString("Content-Transfer-Encoding: quoted-printable\n\n")
	buffer.WriteString(l.content)
	buffer.WriteString(fmt.Sprintf("\n\n--%s\n", boundary))
	l.body = buffer.Bytes()
}

// func (l *Letter) buildAttachments() {
// 	buffer := bytes.NewBuffer(l.body)
// 	for i, item := range l.attachments {
// 		buffer.WriteString("Content-Type: application/octet-stream; charset=UTF-8;\n 	name=\"?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(item.Filename)) + "?=\"\n")
// 		buffer.WriteString("Content-Transfer-Encoding: base64\n")
// 		buffer.WriteString("Content-Disposition: attachment; filename*=utf-8''" + url.PathEscape(item.Filename) + "\n\n")
// 		encodeBuffer := make([]byte, base64.StdEncoding.EncodedLen(len(item.Data)))
// 		base64.StdEncoding.Encode(encodeBuffer, item.Data)
// 		buffer.Write(encodeBuffer)

// 		if i != len(l.attachments)-1 {
// 			buffer.WriteString(fmt.Sprintf("\n--%s\n", boundary))
// 		}
// 	}
// 	l.body = buffer.Bytes()
// }

func (l *Letter) buildEnd() {
	buffer := bytes.NewBuffer(l.body)
	buffer.WriteString("\n--" + boundary + "--\n")
	l.body = buffer.Bytes()
}

// From: https://github.com/jordan-wright/email/blob/master/email.go
// base64Wrap encodes the attachment content, and wraps it according to RFC 2045 standards (every 76 chars)
// The output is then written to the specified io.Writer
func base64Wrap(w io.Writer, b []byte) {
	// 57 raw bytes per 76-byte base64 line.
	const maxRaw = 57
	// Buffer for each line, including trailing CRLF.
	buffer := make([]byte, MaxLineLength+len("\r\n"))
	copy(buffer[MaxLineLength:], "\r\n")
	// Process raw chunks until there's no longer enough to fill a line.
	for len(b) >= maxRaw {
		base64.StdEncoding.Encode(buffer, b[:maxRaw])
		w.Write(buffer)
		b = b[maxRaw:]
	}
	// Handle the last chunk of bytes.
	if len(b) > 0 {
		out := buffer[:base64.StdEncoding.EncodedLen(len(b))]
		base64.StdEncoding.Encode(out, b)
		out = append(out, "\r\n"...)
		w.Write(out)
	}
}

func (l *Letter) buildAttachments() {
	buffer := bytes.NewBuffer(l.body)
	for i, item := range l.attachments {
		base64Name := base64.StdEncoding.EncodeToString([]byte(item.Filename))
		buffer.WriteString("Content-Type: application/octet-stream; charset=UTF-8;\n 	name=\"=?UTF-8?B?" + base64Name + "?=\"\n")
		buffer.WriteString("Content-Transfer-Encoding: base64\n")
		buffer.WriteString("Content-Disposition: attachment; filename=\"=?UTF-8?B?" + base64Name + "?=\"\n\n")

		// encodeBuffer := make([]byte, base64.StdEncoding.EncodedLen(len(item.Data)))
		// base64.StdEncoding.Encode(encodeBuffer, item.Data)
		// buffer.Write(encodeBuffer)
		base64Wrap(buffer, item.Data)

		if i != len(l.attachments)-1 {
			buffer.WriteString(fmt.Sprintf("--%s\n", boundary))
		}
	}
	l.body = buffer.Bytes()
}

// From: github.com/stvoidit/gosmtp/Messanger.go
func generateBoundary() string {
	var buf = make([]byte, 16)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func main() {
	flag.Usage = func() {
		fmt.Println("# Version:", VerStr)
		fmt.Println("# Usage:", os.Args[0], "[args] [attachment.7z]")
		flag.PrintDefaults()
		fmt.Println("# Example:")
		fmt.Println("  [set|export] eSvrSMTP=" + DefServerStr)
		fmt.Println("  [set|export] eUP=xx@163.com:hello")
		fmt.Println("  ", os.Args[0], "[-p] xx.7z [yy.zip]   发送xx.7z [yy.zip]")
		os.Exit(0)
	}
	bPrintEml := false
	flag.BoolVar(&bPrintEml, "p", bPrintEml, "显示邮件")
	flag.Parse()             // 处理参数
	fileCount := flag.NArg() // 处理后的参数个数，一般是URL
	if 0 == fileCount {
		flag.Usage()
	}

	envSV := os.Getenv("eSvrSMTP")
	envUP := os.Getenv("eUP")
	if "" == envUP {
		envUP = "hello@163.com:helloworld"
	}
	aUP := strings.Split(envUP, ":")
	if "" == aUP[0] || "" == aUP[1] {
		os.Exit(1)
	}
	if "" == envSV {
		envSV = DefServerStr
	}
	aEml := strings.Split(envSV, ":")
	if "" == aEml[0] || "" == aEml[1] {
		envSV = DefServerStr
		aEml = strings.Split(envSV, ":")
	}
	nPort, err := strconv.Atoi(aEml[1])
	if err != nil {
		fmt.Println("- Error:", err)
		os.Exit(2)
	}

	sender := NewSender(aEml[0], nPort, "you", aUP[0], aUP[1])

	letterInstance := sender.NewLetter()
	letterInstance.AddReceivers(aUP[0])

	sName := ""
	for n := 0; n < fileCount; n++ {
		attachPathA := flag.Arg(n)
		attachNameA := filepath.Base(attachPathA)
		if 0 == n {
			sName = attachNameA
		} else {
			sName = sName + ":" + attachNameA
		}

		attachment1, err := os.ReadFile(attachPathA)
		if err != nil {
			fmt.Println("- Error:", err)
			os.Exit(3)
		}
		letterInstance.AddAttachment(attachNameA, attachment1)
	}
	letterInstance.SetSubject(sName)
	letterInstance.SetContent(sName)

	if bPrintEml {
		letterInstance.Dump() // 打印邮件体
	}

	err = sender.Send(letterInstance)
	if err != nil {
		fmt.Println("- Error:", err)
		os.Exit(4)
	}
}
