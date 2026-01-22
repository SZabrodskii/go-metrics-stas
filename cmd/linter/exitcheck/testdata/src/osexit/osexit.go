// Package osexit содержит тестовые кейсы для проверки детекции os.Exit.
package osexit

import "os"

func testOsExit() {
	os.Exit(1)
}

func testOsExitZero() {
	os.Exit(0)
}

func testOsGetenv() {
	_ = os.Getenv("HOME")
}
