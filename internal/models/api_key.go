package models

import "time"

// APIKey represents an organization-linked API key 
type APIKey struct {
	Base
	KeyValue       string     `json:"key_value" gorm:"type:varchar(255);uniqueIndex;not null"`
	Description    string     `json:"description" gorm:"type:text"`
	OrganizationID string     `json:"organization_id" gorm:"type:varchar(36);index;not null"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" gorm:"type:timestamp"`

	// Relationships
	Organization Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
}

// TableName specifies the table name for APIKey
func (APIKey) TableName() string {
	return "api_keys"
}