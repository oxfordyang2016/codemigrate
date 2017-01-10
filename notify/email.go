package notify

import (
	"./../utils/cache"
	"bytes"
	"crypto/tls"
	"cydex"
	"encoding/json"
	"errors"
	"fmt"
	clog "github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"html/template"
	"io"
	"net"
	"net/smtp"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultEmailTemplateDir  = "/opt/cydex/etc/ts.d/email/templates"
	DefaultMaxEmailProcesser = 20
	ContentType              = "text/html; charset=UTF-8"
)

var (
	Email *EmailHandler

	HeadersOrder = []string{"To", "From", "Subject", "Content-Type"}
	TplFuncs     = template.FuncMap{"DatetimeFormat": DatetimeFormat}
	SmtpLabels   = []string{"default", "1", "2", "3"}
)

func init() {
	Email = NewEmailHandler()
}

// smtp服务器
type SmtpServer cydex.EmailSmtpServer

type PkgEvMailContext struct {
	TargetUser *cydex.User // 邮件发送目标用户
	PkgEvent   *PkgEvent
}

func NewPkgEvMailContext(pkg_ev *PkgEvent) *PkgEvMailContext {
	if pkg_ev == nil {
		return nil
	}
	o := new(PkgEvMailContext)
	o.PkgEvent = pkg_ev

	// NOTE for test
	// pkg_ev.User = &cydex.User{
	// 	Username:     "receiver",
	// 	Email:        "40744134@qq.com",
	// 	EmailAffairs: 0xff,
	// 	Language:     0,
	// }
	// pkg_ev.Owner = &cydex.User{
	// 	Username:     "sender",
	// 	Email:        "40744134@qq.com",
	// 	EmailAffairs: 0xff,
	// 	Language:     1,
	// }
	// if pkg_ev.Job.Type == cydex.UPLOAD {
	// 	pkg_ev.User = pkg_ev.Owner
	// }
	// end for test

	switch pkg_ev.NotifyType {
	case cydex.NotifyToReceiverNewPkg, cydex.NotifyToReceiverPkgDownloadFinish:
		o.TargetUser = pkg_ev.User
	case cydex.NotifyToSenderPkgDownloadStart, cydex.NotifyToSenderPkgDownloadFinish, cydex.NotifyToSenderPkgUploadFinish:
		o.TargetUser = pkg_ev.Owner
	}

	return o
}

// Email模板管理
type EmailTemplate struct {
	BaseDir      string
	Lang         int
	lock         sync.RWMutex
	subject_tpls map[int]*template.Template // int:NotifyType
	content_tpls map[int]*template.Template // int:NotifyType
}

func NewEmailTemplate(base_dir string) *EmailTemplate {
	o := new(EmailTemplate)
	o.BaseDir = base_dir
	o.subject_tpls = make(map[int]*template.Template)
	o.content_tpls = make(map[int]*template.Template)
	o.Lang = -1
	return o
}

// 按照语言导入模板
func (self *EmailTemplate) LoadByLang(lang_idx int, reload bool) error {
	if lang_idx == self.Lang && !reload {
		return nil
	}

	lang := cydex.LanguageStrs[lang_idx]
	clog.Infof("[mail handler] load templates lang:%s reload:%t", lang, reload)

	self.lock.Lock()
	defer self.lock.Unlock()

	file := filepath.Join(self.BaseDir, "content_base.tpl")
	base, err := template.ParseFiles(file)
	if err != nil {
		return err
	}
	base = base.Funcs(TplFuncs)

	if err := self.loadSubject(lang); err != nil {
		return err
	}
	if err := self.loadContent(lang, base); err != nil {
		return err
	}
	self.Lang = lang_idx
	return nil
}

func (self *EmailTemplate) loadSubject(lang string) error {
	var err error
	for _, v := range NotifyEvents {
		file := filepath.Join(self.BaseDir, lang, fmt.Sprintf("subject_%d.tpl", v))
		self.subject_tpls[v], err = template.ParseFiles(file)
		if err != nil {
			clog.Errorf("EmailTemplate load subject failed, lang:%s, ev:%d, err:%s", lang, v, err.Error())
			return err
		}
	}
	return nil
}

func (self *EmailTemplate) loadContent(lang string, base *template.Template) error {
	var err error
	for _, v := range NotifyEvents {
		file := filepath.Join(self.BaseDir, lang, fmt.Sprintf("content_%d.tpl", v))
		if base != nil {
			self.content_tpls[v], err = template.Must(base.Clone()).ParseFiles(file)
		} else {
			self.content_tpls[v], err = template.ParseFiles(file)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// 渲染标题
func (self *EmailTemplate) RenderSubject(wr io.Writer, ctx *PkgEvMailContext) error {
	if wr == nil || ctx == nil {
		return errors.New("Value Error")
	}
	self.lock.RLock()
	defer self.lock.RUnlock()

	tpl := self.subject_tpls[ctx.PkgEvent.NotifyType]
	if tpl == nil {
		return fmt.Errorf("No subject template defined for notify:%d", ctx.PkgEvent.NotifyType)
	}
	return tpl.Execute(wr, ctx)
}

// 渲染内容
func (self *EmailTemplate) RenderContent(wr io.Writer, ctx *PkgEvMailContext) error {
	if wr == nil || ctx == nil {
		return errors.New("Value Error")
	}
	self.lock.RLock()
	defer self.lock.RUnlock()

	tpl := self.content_tpls[ctx.PkgEvent.NotifyType]
	if tpl == nil {
		return fmt.Errorf("No content template defined for notify:%d", ctx.PkgEvent.NotifyType)
	}
	return tpl.Execute(wr, ctx)
}

// type SmtpServer struct {
// 	cydex.EmailSmtpServer
// 	// Host     string // smtp服务地址
// 	// Port     int    // smtp服务端口
// 	// Account  string // 账号
// 	// Password string // 密码
// 	// UseTLS   bool   // 是否使用TLS连接
// }

type EmailHandler struct {
	ContactName string // 联系人名字, 发件人的名字, 如为空，取Account
	Enable      bool
	templates   map[int]*EmailTemplate

	tpl_base_dir     string
	smtp_server      *SmtpServer
	pkg_ev_queue     chan *PkgEvent
	max_ev_processer int
	lock             sync.RWMutex
}

func NewEmailHandler() *EmailHandler {
	return NewEmailHandlerEx("", "", 0)
}

func NewEmailHandlerEx(tpl_base_dir string, contact_name string, max_processer int) *EmailHandler {
	o := new(EmailHandler)
	o.templates = make(map[int]*EmailTemplate)
	o.pkg_ev_queue = make(chan *PkgEvent)
	if max_processer <= 0 {
		max_processer = DefaultMaxEmailProcesser
	}
	o.max_ev_processer = max_processer
	if contact_name == "" {
		contact_name = "cydex_noreply"
	}
	o.ContactName = contact_name
	o.SetTemplateBaseDir(tpl_base_dir)
	o.Enable = false

	return o
}

func (self *EmailHandler) SetEnable(v bool) {
	self.Enable = v
	clog.Infof("[mail handler] set enable: %t", v)
}

func (self *EmailHandler) SetSmtpServer(cfg *SmtpServer) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.smtp_server = cfg
	clog.Infof("[mail handler] set smtp server: %+v", cfg)
	return nil
}

func (self *EmailHandler) SmtpServer() *SmtpServer {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return self.smtp_server
}

func (self *EmailHandler) SetContactName(name string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.ContactName = name
	clog.Infof("[mail handler] set contact name: %s", name)
}

func (self *EmailHandler) SetTemplateBaseDir(base_dir string) {
	if base_dir == "" {
		base_dir = DefaultEmailTemplateDir
	}

	self.lock.Lock()
	defer self.lock.Unlock()
	self.tpl_base_dir = base_dir
}

func (self *EmailHandler) LoadTemplates() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	basedir := self.tpl_base_dir
	for i := 0; i < cydex.LanguageLast; i++ {
		tpl := NewEmailTemplate(basedir)
		if err := tpl.LoadByLang(i, true); err != nil {
			continue
		}
		self.templates[i] = tpl
	}
	if len(self.templates) == 0 {
		return errors.New("No templates loaded")
	}
	return nil
}

func (self *EmailHandler) Start() {
	go self.ServeSubscribe()
	go self.ServeEvent()
}

func (self *EmailHandler) ServeSubscribe() {
	psc := redis.PubSubConn{cache.Get()}
	psc.Subscribe(PubSubChanForPkg)

	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			self.handleRedisMsg(&v)
		case redis.Subscription:
		case error:
			clog.Error("Email handle redis subscribe error")
		}
	}
}

func (self *EmailHandler) ServeEvent() {
	sem := make(chan int, self.max_ev_processer)
	for ev := range self.pkg_ev_queue {
		ev := ev
		sem <- 1
		go func() {
			self.handlePkgEvent(ev)
			<-sem
		}()
	}
}

func (self *EmailHandler) handleRedisMsg(msg *redis.Message) error {
	if !self.Enable {
		return nil
	}

	switch msg.Channel {
	case PubSubChanForPkg:
		pkg_ev := new(PkgEvent)
		if err := json.Unmarshal(msg.Data, pkg_ev); err != nil {
			return err
		}
		clog.Tracef("[mail handler] handle redis msg, %+v", pkg_ev)
		self.pkg_ev_queue <- pkg_ev
	default:
		return nil
	}
	return nil
}

// 根据lang_idx获取模板，fallback表示如果失败，使用en
func (self *EmailHandler) getTemplate(lang_idx int, fallback bool) *EmailTemplate {
	var tpl *EmailTemplate
	tpl, _ = self.templates[lang_idx]
	if tpl == nil && fallback {
		lang_idx = 0
		tpl, _ = self.templates[lang_idx]
	}
	return tpl
}

func (self *EmailHandler) handlePkgEvent(pkg_ev *PkgEvent) {
	if self.smtp_server == nil {
		return
	}

	mail_ctx := NewPkgEvMailContext(pkg_ev)
	if mail_ctx == nil || mail_ctx.TargetUser == nil || mail_ctx.TargetUser.Email == "" {
		clog.Errorf("[mail handler] Invalid mail context")
		return
	}
	if mail_ctx.TargetUser.EmailAffairs&(1<<uint(pkg_ev.NotifyType)) == 0 {
		clog.Warnf("[mail handler] user %s (mask:0x%x), is not registed this event: %d", mail_ctx.TargetUser.Username, mail_ctx.TargetUser.EmailAffairs, pkg_ev.NotifyType)
		return
	}

	self.lock.RLock()
	smtp_server := *self.smtp_server
	tpl := self.getTemplate(mail_ctx.TargetUser.Language, true)
	self.lock.RUnlock()

	if tpl == nil {
		clog.Errorf("[mail handler] no template found by lang:%d", mail_ctx.TargetUser.Language)
		return
	}

	// render subject and content
	var (
		subject bytes.Buffer
		content bytes.Buffer
	)
	if err := tpl.RenderSubject(&subject, mail_ctx); err != nil {
		clog.Error("render subject ", err)
		return
	}
	if err := tpl.RenderContent(&content, mail_ctx); err != nil {
		clog.Error("render content ", err)
		return
	}

	// prepare
	from := self.ContactName
	if from == "" {
		from = smtp_server.Account
	}

	message := ""

	headers := make(map[string]string)
	headers["To"] = fmt.Sprintf("%s<%s>", mail_ctx.TargetUser.Username, mail_ctx.TargetUser.Email)
	headers["From"] = fmt.Sprintf("%s<%s>", from, smtp_server.Account)
	headers["Subject"] = strings.Trim(subject.String(), "\r\n")
	headers["Content-Type"] = ContentType

	// Setup message
	for _, header := range HeadersOrder {
		message += fmt.Sprintf("%s: %s\r\n", header, headers[header])
	}
	message += "\r\n" + content.String()

	auth := smtp.PlainAuth("", smtp_server.Account, smtp_server.Password, smtp_server.Host)
	addr := fmt.Sprintf("%s:%d", smtp_server.Host, smtp_server.Port)

	// print(message)

	clog.Infof("[mail handler] send mail to %s", mail_ctx.TargetUser.Username)

	// send mail
	if err := SendMail(smtp_server.UseTLS, addr, auth, smtp_server.Account, []string{mail_ctx.TargetUser.Email}, []byte(message)); err != nil {
		clog.Errorf("[mail handler] send mail to %s failed: %s", mail_ctx.TargetUser.Username, err.Error())
	}

	clog.Infof("[mail handler] send mail to %s ok", mail_ctx.TargetUser.Username)
}

// modified from net/smtp/smtp.go, add TLS support
func SendMail(use_tls bool, addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	var err error
	var c *smtp.Client

	host, _, _ := net.SplitHostPort(addr)

	if use_tls {
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		}
		conn, err := tls.Dial("tcp", addr, tlsconfig)
		if err != nil {
			return err
		}

		c, err = smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
	} else {
		c, err = smtp.Dial(addr)
		if err != nil {
			return err
		}
	}
	defer c.Close()
	// if err = c.hello(); err != nil {
	// 	return err
	// }
	// if ok, _ := c.Extension("STARTTLS"); ok {
	// 	config := &tls.Config{ServerName: c.serverName}
	// 	if testHookStartTLS != nil {
	// 		testHookStartTLS(config)
	// 	}
	// 	if err = c.StartTLS(config); err != nil {
	// 		return err
	// 	}
	// }
	if err = c.Auth(a); err != nil {
		return err
	}

	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
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
	return c.Quit()
}

func DatetimeFormat(lang string, t time.Time) string {
	t = t.UTC()
	if lang == "zh" {
		// 中文固定为北京时间
		const layout = "2006-01-02 15:04:05"
		t = t.Add(8 * time.Hour)
		return t.Format(layout)
	} else {
		// 英文为UTC时间
		const layout = "3:04pm Jan 02, 2006"
		return t.Format(layout)
	}
}
