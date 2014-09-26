package main

import (
	"io"
	"log"
	"net"
	"time"
	"regexp"
	"strconv"
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
			log.Print("Temperature is ", v)

		case v := <- sivert :
 			log.Print("Radiation is ", v)
		}
	}
}
