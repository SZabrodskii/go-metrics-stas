// Package main содержит тестовые кейсы для проверки исключения main().
package main

import (
	"log"
	"os"
)

func main() {
	// В main() разрешено использовать os.Exit и log.Fatal
	log.Print("starting")
	os.Exit(0)
}

func otherFunc() {
	// Вне main() запрещено
	log.Fatal("error") // want "call to log.Fatal outside of main\\(\\) in main package"
	os.Exit(1)         // want "call to os.Exit outside of main\\(\\) in main package"
}

func anotherFunc() {
	// panic всегда запрещен, даже в main пакете
	panic("error") // want "usage of builtin panic is discouraged"
}
