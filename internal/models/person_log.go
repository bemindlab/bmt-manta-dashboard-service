package models

import "time"

// PersonLog represents detection of a person by a camera
type PersonLog struct {
	Base
	Timestamp      time.Time `json:"timestamp" gorm:"type:timestamp;index;not null"`
	PersonHash     string    `json:"person_hash" gorm:"type:varchar(255);index;not null"`
	CameraID       string    `json:"camera_id" gorm:"type:varchar(36);index;not null"`
	IsNewPerson    bool      `json:"is_new_person" gorm:"type:boolean;not null;default:false"`
	OrganizationID string    `json:"organization_id" gorm:"type:varchar(36);index;not null"`

	// Relationships
	Camera       Camera       `json:"camera,omitempty" gorm:"foreignKey:CameraID"`
	Organization Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	Person       Person       `json:"person,omitempty" gorm:"foreignKey:PersonHash;references:PersonHash"`
}

// TableName specifies the table name for PersonLog
func (PersonLog) TableName() string {
	return "person_logs"
}

// LogFilter is used for filtering person logs
type LogFilter struct {
	From           time.Time `json:"from,omitempty"`
	To             time.Time `json:"to,omitempty"`
	CameraID       string    `json:"camera_id,omitempty"`
	PersonID       string    `json:"person_id,omitempty"`
	OrganizationID string    `json:"organization_id,omitempty"`
	Page           int       `json:"page,omitempty"`
	PageSize       int       `json:"page_size,omitempty"`
}