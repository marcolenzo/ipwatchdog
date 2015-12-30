package main

import (
	"flag"
	"fmt"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
	"net/smtp"
)

var lastIp, from, to, server_host, server_port, user, passwd, schedule, checkip_url, callback_url, callback_ip_param, callback_auth_header string

func main() {
	flag.StringVar(&from, "from", "", "Sender email address, e.g. user@domain.com")
	flag.StringVar(&to, "to", "", "Recipient email address, e.g. recipient@domain.com.")
	flag.StringVar(&server_host, "server_host", "smtp.gmail.com", "Mail server hostname.")
	flag.StringVar(&server_port, "server_port", "587", "Mail server port.")
	flag.StringVar(&user, "user", "", "Username @ Mail server. If not specified the sender email address is used.")
	flag.StringVar(&passwd, "passwd", "", "Password @ Mail server.")
	flag.StringVar(&schedule, "schedule", "@every 30m", "Schedule defining the time interval between each IP check.")
	flag.StringVar(&checkip_url, "checkip_url", "http://checkip.amazonaws.com", "Check IP API URL.")
	flag.StringVar(&callback_url, "callback_url", "", "URL to hit when IP changes. If not set, callback won't be performed.")
	flag.StringVar(&callback_ip_param, "callback_ip_param", "", "Query parameter to be used to communicate new IP. If left empty IP won't be set.")
	flag.StringVar(&callback_auth_header, "callback_auth_header", "", "Authorization header to be used in callback.")
	flag.Parse()
	if from == "" || to == "" {
		panic("-from and -to parameters are required.")
	}
	initialize()
}

func initialize() {
	c := cron.New()
	c.AddFunc(schedule, func() { checkIp() })
	c.Start()
	fmt.Println("IP Watchdog Cron started...")
	select {}
}

func checkIp() {
	resp, err := http.Get(checkip_url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	currentIp := string(body)
	fmt.Println("Current IP: " + currentIp)
	if lastIp != "" && lastIp != currentIp {
		message := "IP has changed! Previous was: " + lastIp
		fmt.Println(message)
		sendMail([]byte("IP has changed! New IP is " + currentIp + " while previous IP was " + lastIp))
		if callback_url != "" {
			callback(currentIp)
		}
	}
	lastIp = currentIp
}

func sendMail(message []byte) {
	auth := smtp.PlainAuth("", from, passwd, server_host)
	fmt.Println("Sending mail to: " + to)
	err := smtp.SendMail(server_host+":"+server_port, auth, from, []string{to}, message)
	if err != nil {
		fmt.Println(err)
	}
}

func callback(ip string) {
	url := callback_url
	if callback_ip_param != "" {
		url = callback_url + callback_ip_param + ip
	}
	fmt.Println("Invoking callback url: " + url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	if callback_auth_header != "" {
		fmt.Println("Setting Authorization Header to: " + callback_auth_header)
		req.Header.Add("Authorization", callback_auth_header)
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
}
