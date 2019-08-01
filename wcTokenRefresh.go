package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	gomail "gopkg.in/gomail.v2"

	"github.com/go-redis/redis"
)

// RefreshOptions 配置项
type RefreshOptions struct {
	RetryTimes         int    // 失败重试次数
	Tick               int    // 刷新时间间隔 单位：s
	WechatAppID        string // APP_ID
	WechatAPPSecret    string // APP_SECRET
	RedisAddr          string // Redis连接地址 ip:port
	RedisPassword      string // Redis连接密码
	RedisDB            int    // Redis数据库索引
	RedisKey           string // Token在redis中的key
	RedisJsKey         string // js ticket 在redis中的key
	MailAlert          bool   // 是否开启邮件告警
	MailSMTPServer     string // 发信邮件服务器
	MailSMTPPort       int    // 发信邮件服务器端口
	MailAccount        string // 发信邮箱
	MailPassword       string // 发信邮箱密码
	MailReceiveAccount string // 告警邮件收信邮箱
}

var wcAccessToken string

func main() {

	// 初始化配置
	options := initOptions()
	if options == nil {
		log.Fatalln("初始化配置出错！")
		return
	}

	tickChan := time.NewTicker(time.Second * time.Duration(options.Tick)).C

	processChan := make(chan int, 1)
	// 启动时向channel输入数据，使得程序立即执行一次获取Token操作
	processChan <- 1

	signalChan := make(chan os.Signal)

	// 监听结束信号
	signal.Notify(signalChan, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTSTP)

	for {
		select {
		case <-processChan:
			isSuccess := false

			// 尝试获取
			for i := 0; i < options.RetryTimes; i++ {
				wxToken := retrieveWechatToken(options)
				// 获取失败，进行下一次尝试
				if len(wxToken) == 0 {
					log.Println("第" + string(i) + "次获取access token失败！进入下一次尝试")
					time.Sleep(time.Second * 5)
					continue
				}

				wcAccessToken = wxToken

				// 发送Redis
				err := sendRedis(wxToken, options, options.RedisKey)
				if err != nil {
					log.Println(err)
					continue
				}

				log.Println("Success Access Token! " + wxToken)
				isSuccess = true

				break
			}

			// 如果获取失败,并且配置了邮件报警则发送邮件
			if !isSuccess && options.MailAlert {
				sendMail(options, fmt.Sprintf("%s %s", "Token获取失败！", time.Now()))
			}

			isSuccess = false

			// 尝试获取js Token
			for i := 0; i < options.RetryTimes; i++ {
				jsTicket := retrieveWechatJsTicket(wcAccessToken)
				// 获取失败，进行下一次尝试
				if len(jsTicket) == 0 {
					log.Println("第" + string(i) + "次获取js ticket失败！进入下一次尝试")
					time.Sleep(time.Second * 5)
					continue
				}

				// 发送Redis
				err := sendRedis(jsTicket, options, options.RedisJsKey)
				if err != nil {
					log.Println(err)
					continue
				}

				log.Println("Success Js Ticket! " + jsTicket)
				isSuccess = true

				break
			}

			// 如果获取失败,并且配置了邮件报警则发送邮件
			if !isSuccess && options.MailAlert {
				sendMail(options, fmt.Sprintf("%s %s", "Ticket获取失败！", time.Now()))
			}

		case <-tickChan:
			processChan <- 1
		case <-signalChan:
			fmt.Println("\nDone!")
			return
		}
	}
}

// 从微信服务器获取js Ticket
func retrieveWechatJsTicket(accessToken string) string {

	url := "https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=" + accessToken + "&type=jsapi"

	// 获取Ticket
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalln(err)
		return ""
	}

	// 解析Resp
	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	var jsonStr map[string]interface{}
	json.Unmarshal(bodyData, &jsonStr)

	// 读取ticket
	jsTicket := jsonStr["ticket"]

	if jsTicket == nil {
		log.Fatalln("Ticket获取失败！")
		return ""
	}

	jsTicketStr, ok := jsTicket.(string)
	if !ok {
		log.Fatalln("Ticket转换失败！")
		return ""
	}

	return jsTicketStr

}

// 从微信服务器获取Token
func retrieveWechatToken(options *RefreshOptions) string {
	// 构造获取Token的URL
	url := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + options.WechatAppID + "&secret=" + options.WechatAPPSecret

	// 获取Token
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalln(err)
		return ""
	}

	// 解析Resp
	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	var jsonStr map[string]interface{}
	json.Unmarshal(bodyData, &jsonStr)

	// 读取access_token
	wxToken := jsonStr["access_token"]

	if wxToken == nil {
		log.Fatalln("Token获取失败！")
		return ""
	}

	wxTokenStr, ok := wxToken.(string)
	if !ok {
		log.Fatalln("Token转换失败！")
		return ""
	}

	return wxTokenStr
}

func sendRedis(value string, options *RefreshOptions, redisKey string) error {

	// 初始化Redis连接
	client := redis.NewClient(&redis.Options{
		Addr:     options.RedisAddr,
		Password: options.RedisPassword,
		DB:       options.RedisDB,
	})

	// 测试Redis连通性
	_, err := client.Ping().Result()
	if err != nil {
		log.Fatalln(err)
		return err
	}

	// 写入Redis
	status := client.Set(redisKey, value, 0)
	if status.Err() != nil {
		log.Fatalln(status.Err())
		return status.Err()
	}

	return nil
}

// 检查并初始化各项配置
func initOptions() *RefreshOptions {
	if len(os.Getenv("REDIS_ADDR")) == 0 {
		log.Fatalln("缺少REDIS_ADDR，请在配置中填写！")
		return nil
	}

	if len(os.Getenv("REDIS_PASSWORD")) == 0 {
		log.Fatalln("缺少REDIS_PASSWORD，请在配置中填写！")
		return nil
	}

	if len(os.Getenv("REDIS_DB")) == 0 {
		os.Setenv("REDIS_DB", "0")
	}

	if len(os.Getenv("REDIS_KEY")) == 0 {
		os.Setenv("REDIS_KEY", "wechat:access_token")
	}

	if len(os.Getenv("REDIS_JS_KEY")) == 0 {
		os.Setenv("REDIS_JS_KEY", "wechat:jsapi_ticket")
	}

	if len(os.Getenv("RETRY_TIMES")) == 0 {
		os.Setenv("RETRY_TIMES", "5")
	}

	if len(os.Getenv("TICK")) == 0 {
		os.Setenv("TICK", "7160")
	}

	if len(os.Getenv("WECHAT_APP_ID")) == 0 {
		log.Fatalln("缺少APP_ID，请在配置中填写！")
		return nil
	}

	if len(os.Getenv("WECHAT_APP_SECRET")) == 0 {
		log.Fatalln("缺少APP_SECRET，请在配置中填写！")
		return nil
	}

	db, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	retryTimes, err := strconv.Atoi(os.Getenv("RETRY_TIMES"))
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	tick, err := strconv.Atoi(os.Getenv("TICK"))
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	mailAlert, err := strconv.ParseBool(os.Getenv("MAIL_ALERT"))
	if err != nil {
		log.Fatalln(err)
		mailAlert = false
	}

	if len(os.Getenv("MAIL_SMTP_SERVER")) == 0 {
		os.Setenv("MAIL_SMTP_SERVER", "")
	}

	smtpPort, err := strconv.Atoi(os.Getenv("MAIL_SMTP_PORT"))
	if err != nil {
		log.Fatalln(err)
		smtpPort = 465
	}

	if len(os.Getenv("MAIL_ACCOUNT")) == 0 {
		os.Setenv("MAIL_ACCOUNT", "")
	}

	if len(os.Getenv("MAIL_PASSWORD")) == 0 {
		os.Setenv("MAIL_PASSWORD", "")
	}

	if len(os.Getenv("MAIL_REVEIVE_ACCOUNT")) == 0 {
		os.Setenv("MAIL_REVEIVE_ACCOUNT", "")
	}

	options := RefreshOptions{
		RetryTimes:         retryTimes,
		Tick:               tick,
		RedisDB:            db,
		RedisAddr:          os.Getenv("REDIS_ADDR"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisKey:           os.Getenv("REDIS_KEY"),
		RedisJsKey:         os.Getenv("REDIS_JS_KEY"),
		WechatAppID:        os.Getenv("WECHAT_APP_ID"),
		WechatAPPSecret:    os.Getenv("WECHAT_APP_SECRET"),
		MailAlert:          mailAlert,
		MailSMTPServer:     os.Getenv("MAIL_SMTP_SERVER"),
		MailSMTPPort:       smtpPort,
		MailAccount:        os.Getenv("MAIL_ACCOUNT"),
		MailPassword:       os.Getenv("MAIL_PASSWORD"),
		MailReceiveAccount: os.Getenv("MAIL_RECEIVE_ACCOUNT"),
	}

	return &options
}

// 发送告警邮件
func sendMail(options *RefreshOptions, mailContent string) {
	m := gomail.NewMessage()
	m.SetAddressHeader("From", options.MailAccount, "admin") // 发件人
	m.SetHeader("To", m.FormatAddress(options.MailReceiveAccount, "admin"))
	m.SetHeader("Subject", "微信Token刷新报警!")                          // 主题
	m.SetBody("text/html", fmt.Sprintf("<h2>%s</h2>", mailContent)) // 正文

	d := gomail.NewPlainDialer(options.MailSMTPServer, options.MailSMTPPort, options.MailAccount, options.MailPassword) // 发送邮件服务器、端口、发件人账号、发件人密码
	if err := d.DialAndSend(m); err != nil {
		log.Fatalln(err)
	}
}
