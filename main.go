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
)

var debug = flag.Bool("debug", false, "Enable debug output")

var tempRegexp = regexp.MustCompile("\\+TEMP: [0-9]+")
var sivertRegexp = regexp.MustCompile("\\+SIVERT: [0-9]+")
var humRegexp = regexp.MustCompile("\\+HUM: [0-9]+")
var pressRegexp = regexp.MustCompile("\\+PRESS: [0-9]+")

func searchForMessage(buffer []byte, pattern * regexp.Regexp, offset int, upstream chan int) {
	match := pattern.Find(buffer)
	if(match != nil) {
		value, err := strconv.Atoi(string(match[offset:]))
		if err == nil {
			upstream <- value
		} else {
			log.Print("Cound not convert value to int.", string(match[offset:]))
		}
	}
}

func splitMessage(con io.Reader, temp chan int, sivert chan int, hum chan int, press chan int){
	buffer := make([]byte, 256)

	for {
		length, err := con.Read(buffer)
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
		message := fmt.Sprintf("%s %f %d\n", id, value, date.Unix())
		if(*debug){
			log.Print(message)
		}
		carbon.Write([]byte(message))
		carbon.Close()
	}
}

func main() {

	flag.Parse()

	con, err := net.Dial("udp", "10.12.20.11:56345")

	timer := time.NewTicker(time.Second * 2)

	temp := make(chan int)
	sivert := make(chan int)
	hum := make(chan int)
	press := make(chan int)

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

		case v := <- temp : 
			value := float64(v)/10.0 
			sendDataToGraphite("hs.hardroom.temperature", value)

		case v := <- sivert :
			value := float64(v)/10000.0 // AS uS/h
			sendDataToGraphite("hs.hardroom.radiation", value)

		case v := <- hum :
			value := float64(v)
			sendDataToGraphite("hs.hardroom.humidity", value)

		case v := <- press :
			value := float64(v)/100 // AS hPa
			sendDataToGraphite("hs.hardroom.pressure", value)

		}
	}
}
