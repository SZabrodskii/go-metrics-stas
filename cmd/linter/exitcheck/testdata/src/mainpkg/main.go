// Package main содержит тестовые кейсы для проверки исключения main().
package main

import (
	"log"
	"os"
)

func main() {
	log.Print("starting")
	os.Exit(0)
}

func otherFunc() {
	// Вне main() запрещено
	log.Fatal("error")
	os.Exit(1)
}

func anotherFunc() {
	// panic всегда запрещен, даже в main пакете
	panic("error") // want "usage of builtin panic is discouraged"
}
