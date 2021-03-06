package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
)

var lastIp, email_sender_address, email_recipient_address, email_server_host, email_server_port, email_server_username, email_server_password, schedule, checkip_url, callback_url, callback_ip_param, callback_auth_header, credentials_file_path string
var email_alert_on, callback_on, schedule_callback_on bool
var reader = bufio.NewReader(os.Stdin)

func main() {
	// TODO modularize this program according to Go best practices.
	flag.StringVar(&email_sender_address, "email_sender_address", "", "Sender email address, e.g. user@domain.com")
	flag.StringVar(&email_recipient_address, "email_recipient_address", "", "Recipient email address, e.g. recipient@domain.com.")
	flag.StringVar(&email_server_host, "email_server_host", "smtp.gmail.com", "Mail server hostname.")
	flag.StringVar(&email_server_port, "email_server_port", "587", "Mail server port.")
	flag.StringVar(&email_server_username, "email_server_username", "", "Username @ Mail server. If not specified the sender email address is used.")
	flag.StringVar(&email_server_password, "email_server_password", "", "Password @ Mail server.")
	flag.StringVar(&schedule, "schedule", "@every 30m", "Schedule defining the time interval between each IP check.")
	flag.StringVar(&checkip_url, "checkip_url", "http://checkip.amazonaws.com", "Check IP API URL.")
	flag.StringVar(&callback_url, "callback_url", "", "URL to hit when IP changes. If not set, callback won't be performed.")
	flag.StringVar(&callback_ip_param, "callback_ip_param", "", "Query parameter to be used to communicate new IP. If left empty IP won't be set.")
	flag.StringVar(&callback_auth_header, "callback_auth_header", "", "Authorization header to be used in callback.")
	flag.StringVar(&credentials_file_path, "credentials_file_path", "/root/go/bin/.ipwatchdog", "Authorization header to be used in callback.")
	flag.BoolVar(&schedule_callback_on, "schedule_callback_on", true, "Callback is invoked at the scheduling interval even if IP did not change")
	flag.Parse()
	if email_sender_address != "" || email_recipient_address != "" || email_server_username != "" || email_server_password != "" {
		email_alert_on = true
		validateEmailSettings()
	}
	if callback_url != "" {
		callback_on = true
		validateCallbackSettings()
	}
	// Try to parse credentials file

	if email_alert_on == false && callback_on == false {
		fmt.Println("Both \"email_alert_on\" and \"callback_on\" are set to false. Exiting...")
		os.Exit(1)
	}

	loadCredentials()
	
	if email_alert_on == true && email_server_password == "" {
		fmt.Println("Enter server password: ")
		email_server_password, _ = reader.ReadString('\n')
	}
	if callback_on == true && callback_auth_header == "" {
		fmt.Println("Enter callback authorization header: ")
		callback_auth_header, _ = reader.ReadString('\n')
	}
	persistCredentials()
	
	initialize()
}

func loadCredentials() {
	f, err := os.Open(credentials_file_path)
	 if err != nil {
		fmt.Println(err)
		return
	}
	fReader := bufio.NewReader(f);
	email_server_password, err = fReader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	callback_auth_header, err = fReader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
}

func persistCredentials() {
	if email_server_password == "" {
		email_server_password = "\n"
	}
	s := []byte(email_server_password + callback_auth_header);
    err := ioutil.WriteFile(credentials_file_path, s, 0600)
    if err != nil {
		fmt.Println(err)
		return
	}
}

func validateEmailSettings() {
	if email_sender_address == "" || email_recipient_address == "" {
		fmt.Println("You need to define \"email_sender_address\" and \"email_recipient_address\" to enable email alerts!")
		os.Exit(1)
	}
}

func validateCallbackSettings() {
	if callback_url == "" {
		fmt.Println("You need to define \"callback_url\" to enable HTTP callbacks")
		os.Exit(1)
	}
}

func initialize() {
	c := cron.New()
	c.AddFunc(schedule, func() { checkIp() })
	// Run immediately once before cron takes over
	checkIp()
	c.Start()
	fmt.Println("IP Watchdog Cron started...")
	select {}
}

func checkIp() {
	resp, err := http.Get(checkip_url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	currentIp := string(body)
	fmt.Println("Current IP: " + currentIp)
	if lastIp != "" && lastIp != currentIp {
		message := "IP has changed! Previous was: " + lastIp
		fmt.Println(message)
		sendMail([]byte("IP has changed! New IP is " + currentIp + " while previous IP was " + lastIp))
		if callback_on && !schedule_callback_on {
			callback(currentIp)
		}
	}
	lastIp = currentIp
	if callback_on && schedule_callback_on {
		callback(currentIp)
	}
}

func sendMail(message []byte) {
	if email_server_username == "" {
		email_server_username = email_sender_address
	}
	auth := smtp.PlainAuth("", email_server_username, email_server_password, email_server_host)
	fmt.Println("Sending mail to: " + email_recipient_address)
	err := smtp.SendMail(email_server_host+":"+email_server_port, auth, email_sender_address, []string{email_recipient_address}, message)
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
		req.Header.Add("Authorization", callback_auth_header)
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("Server replied: " + string(body))
}
