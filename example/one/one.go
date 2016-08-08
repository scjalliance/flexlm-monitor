package main

import (
	"log"
	"time"

	"github.com/scjalliance/flexlm"
)

func main() {
	log.Fatalln(flexlm.TailLog("test.log", time.UTC))
}
