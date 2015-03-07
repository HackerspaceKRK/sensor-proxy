package main

import (
	"io"
	"log"
	"net"
	"time"
	"regexp"
	"strconv"
	"fmt"
	"flag"
	"net/http"
	"io/ioutil"
)

var debug = flag.Bool("debug", false, "Enable debug output")
var delay = flag.Int("delay", 10, "delay between sensor reads")

var tempRegexp = regexp.MustCompile("\\+TEMP: [0-9.]+")
var sivertRegexp = regexp.MustCompile("\\+SIVERT: [0-9.]+")
var humRegexp = regexp.MustCompile("\\+HUM: [0-9.]+")
var pressRegexp = regexp.MustCompile("\\+PRESS: [0-9.]+")

func searchForMessage(buffer []byte, pattern * regexp.Regexp, offset int, upstream chan float64) {
	match := pattern.Find(buffer)
	if(match != nil) {
		value, err := strconv.ParseFloat(string(match[offset:]), 64)
		if err == nil {
			upstream <- value
		} else {
			log.Print("Cound not convert value to float.", string(match[offset:]))
		}
	}
}

func splitMessage(con io.Reader, temp chan float64, sivert chan float64, hum chan float64, press chan float64){
	buffer := make([]byte, 256)

	for {
		length, err := con.Read(buffer)
		if(*debug){
			log.Print(string(buffer))
		}
		if (err != nil) {
			log.Fatal(err)
		}
		if(length > 0){
			searchForMessage(buffer, tempRegexp, 7, temp)
			searchForMessage(buffer, sivertRegexp, 9, sivert)
			searchForMessage(buffer, humRegexp, 6, hum)
			searchForMessage(buffer, pressRegexp, 8, press)
		}
	}
}

func sendDataToGraphite(id string, value float64){
	carbon, err := net.Dial("tcp", "graphite.at.hskrk.pl:2003")
	if err == nil {
		date := time.Now()
		message := fmt.Sprintf("%s %.16f %d\n", id, value, date.Unix())
		if(*debug){
			log.Print(message)
		}
		carbon.Write([]byte(message))
		carbon.Close()
	}
}

func handleKdHomeTemperature(){
	resp, err := http.Get("http://al2.hskrk.pl/api/v2/temp/get")
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	value, err := strconv.ParseFloat(string(body), 64)
	if(err == nil){
		sendDataToGraphite("hs.korytarzyk.temperature", value)
	} else {
		log.Print("Erro on hd home", err)
	}
}

func main() {

	flag.Parse()

	con, err := net.Dial("udp", "10.12.20.11:56345")

	timer := time.NewTicker(time.Second * time.Duration(*delay))

	temp := make(chan float64)
	sivert := make(chan float64)
	hum := make(chan float64)
	press := make(chan float64)	

	if(err != nil){
		log.Fatal("Could not connect to sensor")
	}

	go splitMessage(con, temp, sivert, hum, press)

	for {

		select {

		case <-timer.C :
			con.Write([]byte("AT+TEMP"))
			con.Write([]byte("AT+SIVERT"))
			con.Write([]byte("AT+HUM"))
			con.Write([]byte("AT+PRESS"))
			go handleKdHomeTemperature()

		case v := <- temp : 
			value := float64(v)
			sendDataToGraphite("hs.hardroom.temperature", value)

		case v := <- sivert :
			value := float64(v) 
			sendDataToGraphite("hs.hardroom.radiation", value)

		case v := <- hum :
			value := float64(v)
			sendDataToGraphite("hs.hardroom.humidity", value)

		case v := <- press :
			value := float64(v)
			sendDataToGraphite("hs.hardroom.pressure", value)

		}
	}
}
