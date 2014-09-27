package main

import (
	"io"
	"log"
	"net"
	"time"
	"regexp"
	"strconv"
	"fmt"
)

var tempRegexp = regexp.MustCompile("\\+TEMP: [0-9]+")

func searchForTemp(buffer []byte, upstream chan int) {
	match := tempRegexp.Find(buffer)
	
	if(match != nil) {
		value, err := strconv.Atoi(string(match[7:]))
		if err == nil {
			upstream <- value
		}
	}
}

var sivertRegexp = regexp.MustCompile("\\+SIVERT: [0-9]+")

func searchForSivert(buffer []byte, upstream chan int) {
	match := sivertRegexp.Find(buffer)
	if(match != nil) {
		value, err := strconv.Atoi(string(match[9:]))
		if err == nil {
			upstream <- value
		}
	}
}


func splitMessage(con io.Reader, temp chan int, sivert chan int){
	buffer := make([]byte, 256)
	for {
		length, err := con.Read(buffer)
		if (err != nil) {
			log.Fatal(err)
		}
		if(length > 0){
			searchForTemp(buffer, temp)
			searchForSivert(buffer, sivert)
		}
	}
}

func sendDataToGraphite(id string, value float64){
	carbon, err := net.Dial("tcp", "graphite.at.hskrk.pl:2003")
	if err == nil {
		date := time.Now()
		message := fmt.Sprintf("%s %f %d\n", id, value, date.Unix())
		log.Print(message)
		carbon.Write([]byte(message))
		carbon.Close()
	}


}

func main() {

	con, err := net.Dial("udp", "10.12.20.11:56345")

	timer := time.NewTicker(time.Second * 2)

	temp := make(chan int)
	sivert := make(chan int)

	if(err != nil){
		log.Fatal("Could not connect to sensor")
	}

	go splitMessage(con, temp, sivert)

	for {

		select {

		case <-timer.C :
			con.Write([]byte("AT+TEMP"))
			con.Write([]byte("AT+SIVERT"))

		case v := <- temp : 
			value := float64(v)/10.0
			sendDataToGraphite("hs.hardroom.temperature", value)

		case v := <- sivert :
			value := float64(v)/10000.0
			sendDataToGraphite("hs.hardroom.radiation", value)

		}
	}
}
