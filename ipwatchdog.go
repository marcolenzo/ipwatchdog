package main

import (
	"fmt"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
)

var lastIp string

func main() {
	c := cron.New()
	c.AddFunc("@every 5m", func() { checkIp() })
	c.Start()
	fmt.Println("IP Watchdog Cron started...")
	select {}
}

func checkIp() {
	resp, err := http.Get("http://checkip.amazonaws.com")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	currentIp := string(body)
	fmt.Println("Current IP: " + currentIp)
	if lastIp != "" && lastIp != currentIp {
		fmt.Println("IP has changed! Previous was: " + lastIp)
		// TODO Implement warning
	}
	lastIp = currentIp
}
