package models

// Organization represents a company or organization that owns cameras
type Organization struct {
	Base
	Name        string `json:"name" gorm:"type:varchar(255);not null"`
	Description string `json:"description" gorm:"type:text"`

	// Relationships with eager loading disabled by default
	Users      []User      `json:"users,omitempty" gorm:"foreignKey:OrganizationID"`
	Cameras    []Camera    `json:"cameras,omitempty" gorm:"foreignKey:OrganizationID"`
	PersonLogs []PersonLog `json:"-" gorm:"foreignKey:OrganizationID"`
	FaceImages []FaceImage `json:"-" gorm:"foreignKey:OrganizationID"`
	APIKeys    []APIKey    `json:"-" gorm:"foreignKey:OrganizationID"`
	Persons    []Person    `json:"-" gorm:"foreignKey:OrganizationID"`
}

// TableName specifies the table name for Organization
func (Organization) TableName() string {
	return "organizations"
}