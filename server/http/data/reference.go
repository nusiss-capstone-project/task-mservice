package data

type DataMetricVO struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

type MetricOperatorVO struct {
	ID      int    `json:"id"`
	Code    string `json:"code"`
	Display string `json:"display"`
}
