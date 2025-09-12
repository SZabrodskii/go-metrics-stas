package main

import (
	"fmt"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"log"
	"net/http"
)

func main() {

	storage := repository.NewMemStorage()

	metricsHandler := handler.NewMetricsHandler(storage)

	http.HandleFunc("/update/", metricsHandler.UpdateMetric)

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("could not start server: %v\n", err)
	}
}
