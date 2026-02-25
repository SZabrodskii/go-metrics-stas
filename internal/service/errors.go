package service

import "errors"

var (
	ErrInvalidMetricType = errors.New("invalid metric type")
	ErrMetricNotFound    = errors.New("metric not found")
	ErrInvalidMetricID   = errors.New("metric id is required")
	ErrMissingValue      = errors.New("value is required for gauge")
	ErrMissingDelta      = errors.New("delta is required for counter")
)
