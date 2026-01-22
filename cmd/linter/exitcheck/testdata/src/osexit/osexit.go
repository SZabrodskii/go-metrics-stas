// Package osexit содержит тестовые кейсы для проверки детекции os.Exit.
package osexit

import "os"

func testOsExit() {
	os.Exit(1) // want "call to os.Exit outside of main\\(\\) in main package"
}

func testOsExitZero() {
	os.Exit(0) // want "call to os.Exit outside of main\\(\\) in main package"
}

func testOsGetenv() {
	// os.Getenv не должен вызывать ошибку
	_ = os.Getenv("HOME")
}
