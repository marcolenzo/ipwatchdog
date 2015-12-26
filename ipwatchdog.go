package main

import (
	"flag"
	"fmt"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
	"net/smtp"
)

var lastIp, from, to, server_host, server_port, user, passwd, schedule, checkip_url string

func main() {
	flag.StringVar(&from, "from", "", "Sender email address, e.g. user@domain.com")
	flag.StringVar(&to, "to", "", "Recipient email address, e.g. recipient@domain.com")
	flag.StringVar(&server_host, "server_host", "smtp.gmail.com", "Mail server hostname.")
	flag.StringVar(&server_port, "server_port", "587", "Mail server port.")
	flag.StringVar(&user, "user", "", "Username @ Mail server. If not specified the sender email address is used.")
	flag.StringVar(&passwd, "passwd", "", "Password @ Mail server.")
	flag.StringVar(&schedule, "schedule", "@every 30min", "Schedule defining the time interval between each IP check.")
	flag.StringVar(&checkip_url, "checkip_url", "http://checkip.amazonaws.com", "Check IP API URL.")
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
