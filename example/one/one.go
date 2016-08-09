package main

import (
	"log"
	"time"

	"github.com/scjalliance/adskflex-monitor"
)

func main() {
	log.Fatalln(flexlm.TailLog("test.log", time.UTC))
}
