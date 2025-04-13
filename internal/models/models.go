// Package models provides all data models for the application
//
// This package contains:
// - Database entity models with GORM annotations
// - Data Transfer Objects (DTOs) for API requests and responses
// - Filter and pagination structures
package models

// Core DB models provided by this package:
// - Base: Common fields for all models
// - Organization: Company/organization that owns cameras
// - User: System user with authentication
// - APIKey: Authentication key for API access
// - Camera: Physical camera device
// - Person: Tracked person with identity
// - PersonLog: Event log for person detection
// - FaceImage: Stored image of a detected face
//
// DTO models for API:
// - DailySummary: Daily statistics about visitors
// - HeatmapData: Time-based density data
// - PersonStats: Statistics about new vs returning visitors
// - LogFilter: Query parameters for filtering logs
// - Pagination: Response structure for paginated results