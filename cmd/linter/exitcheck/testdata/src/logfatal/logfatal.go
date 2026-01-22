// Package logfatal содержит тестовые кейсы для проверки детекции log.Fatal.
package logfatal

import "log"

func testLogFatal() {
	log.Fatal("fatal error")
}

func testLogFatalf() {
	log.Fatalf("fatal error: %v", "error")
}

func testLogFatalln() {
	log.Fatalln("fatal error")
}

func testLogPrint() {
	log.Print("normal log")
}
