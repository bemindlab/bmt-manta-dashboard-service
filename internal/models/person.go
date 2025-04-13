package models

import "time"

// Person represents a tracked person entity with multiple face images
type Person struct {
	Base
	PersonHash     string      `json:"person_hash" gorm:"type:varchar(255);uniqueIndex;not null"`
	FirstSeen      time.Time   `json:"first_seen" gorm:"type:timestamp;not null"`
	LastSeen       time.Time   `json:"last_seen" gorm:"type:timestamp;not null"`
	VisitCount     int         `json:"visit_count" gorm:"type:int;not null;default:0"`
	OrganizationID string      `json:"organization_id" gorm:"type:varchar(36);index;not null"`
	
	// Relationships
	Organization Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	FaceImages   []FaceImage  `json:"face_images,omitempty" gorm:"foreignKey:PersonHash;references:PersonHash"`
	PersonLogs   []PersonLog  `json:"person_logs,omitempty" gorm:"foreignKey:PersonHash;references:PersonHash"`
}

// TableName specifies the table name for Person
func (Person) TableName() string {
	return "persons"
}