package kpm

import "time"

// Minimal KPM sample (aligned概念：E2SM-KPM 提供KPM回報；此處僅MVP輸出結構)
type SamplePoint struct {
Metric string    `json:"metric"`
Value  float64   `json:"value"`
TS     time.Time `json:"ts"`
}

func Sample(metric string, v float64) SamplePoint {
if v < 0 { v = 0 }
if v > 1 { v = 1 }
return SamplePoint{Metric: metric, Value: v, TS: time.Now().UTC()}
}
