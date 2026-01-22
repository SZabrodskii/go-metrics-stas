// Package logfatal содержит тестовые кейсы для проверки детекции log.Fatal.
package logfatal

import "log"

func testLogFatal() {
	log.Fatal("fatal error") // want "call to log.Fatal outside of main\\(\\) in main package"
}

func testLogFatalf() {
	log.Fatalf("fatal error: %v", "error") // want "call to log.Fatalf outside of main\\(\\) in main package"
}

func testLogFatalln() {
	log.Fatalln("fatal error") // want "call to log.Fatalln outside of main\\(\\) in main package"
}

func testLogPrint() {
	// Обычный log.Print не должен вызывать ошибку
	log.Print("normal log")
}
