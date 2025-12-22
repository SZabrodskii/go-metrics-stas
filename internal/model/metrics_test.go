package model

import "testing"

func TestGetMetricType(t *testing.T) {
	tests := []struct {
		name         string
		metricName   string
		expectedType string
	}{
		{
			name:         "Counter metric PollCount",
			metricName:   MetricPollCount,
			expectedType: Counter,
		},
		{
			name:         "Gauge metric Alloc",
			metricName:   MetricAlloc,
			expectedType: Gauge,
		},
		{
			name:         "Unknown metric defaults to gauge",
			metricName:   "UnknownMetric",
			expectedType: Gauge,
		},
		{
			name:         "Empty string defaults to gauge",
			metricName:   "",
			expectedType: Gauge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMetricType(tt.metricName)
			if result != tt.expectedType {
				t.Errorf("Expected %s, got %s", tt.expectedType, result)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if Counter != "counter" {
		t.Errorf("Expected Counter to be 'counter', got %s", Counter)
	}
	if Gauge != "gauge" {
		t.Errorf("Expected Gauge to be 'gauge', got %s", Gauge)
	}
}

func TestMetricsSlices(t *testing.T) {
	if len(GaugeMetrics) == 0 {
		t.Error("GaugeMetrics should not be empty")
	}
	if len(CounterMetrics) == 0 {
		t.Error("CounterMetrics should not be empty")
	}
	
	expectedTotal := len(GaugeMetrics) + len(CounterMetrics)
	if len(AllMetrics) != expectedTotal {
		t.Errorf("Expected AllMetrics length %d, got %d", expectedTotal, len(AllMetrics))
	}
}