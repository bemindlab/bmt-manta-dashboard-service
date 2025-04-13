package models

// User represents a system user
type User struct {
	Base
	FirebaseUID    string `json:"firebase_uid" gorm:"type:varchar(128);uniqueIndex"`
	Email          string `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Name           string `json:"name" gorm:"type:varchar(255);not null"`
	Role           string `json:"role" gorm:"type:varchar(50);not null"`
	OrganizationID string `json:"organization_id" gorm:"type:varchar(36);index;not null"`

	// Relationships
	Organization Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}