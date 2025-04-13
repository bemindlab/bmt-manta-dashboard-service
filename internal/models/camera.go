package models

// Camera represents a physical camera device
type Camera struct {
	Base
	Name           string `json:"name" gorm:"type:varchar(255);not null"`
	Location       string `json:"location" gorm:"type:varchar(255)"`
	Status         string `json:"status" gorm:"type:varchar(50);not null;default:'active'"`
	OrganizationID string `json:"organization_id" gorm:"type:varchar(36);index;not null"`

	// Relationships
	Organization Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	PersonLogs   []PersonLog  `json:"person_logs,omitempty" gorm:"foreignKey:CameraID"`
	FaceImages   []FaceImage  `json:"face_images,omitempty" gorm:"foreignKey:CameraID"`
}

// TableName specifies the table name for Camera
func (Camera) TableName() string {
	return "cameras"
}