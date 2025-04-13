package models

// DailySummary represents a daily summary of people counts
type DailySummary struct {
	Date           string `json:"date"`
	Total          int    `json:"total"`
	New            int    `json:"new"`
	Repeat         int    `json:"repeat"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// HeatmapData represents density data by time period
type HeatmapData struct {
	Hour           string `json:"hour"`
	Count          int    `json:"count"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// PersonStats represents statistics about new vs returning people
type PersonStats struct {
	New            int    `json:"new"`
	Repeat         int    `json:"repeat"`
	OrganizationID string `json:"organization_id,omitempty"`
}