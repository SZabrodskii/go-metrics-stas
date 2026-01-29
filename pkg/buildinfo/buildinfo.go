// Package buildinfo содержит информацию о сборке приложения.
// Значения устанавливаются при компиляции через ldflags.
package buildinfo

import "fmt"

// Переменные сборки, устанавливаемые через ldflags:
// go build -ldflags "-X pkg/buildinfo.Version=1.0.0 -X pkg/buildinfo.Date=2024-01-01 -X pkg/buildinfo.Commit=abc123"
var (
	Version = "N/A"
	Date    = "N/A"
	Commit  = "N/A"
)

// Print в stdout.
func Print() {
	fmt.Printf("Build version: %s\n", Version)
	fmt.Printf("Build date: %s\n", Date)
	fmt.Printf("Build commit: %s\n", Commit)
}

// String возвращает информацию о сборке в виде строки.
func String() string {
	return fmt.Sprintf("version=%s, date=%s, commit=%s", Version, Date, Commit)
}
