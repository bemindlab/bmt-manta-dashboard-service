package models

// FaceImage represents a stored image of a detected face
type FaceImage struct {
	Base
	PersonHash     string `json:"person_hash" gorm:"type:varchar(255);index;not null"`
	ImageURL       string `json:"image_url" gorm:"type:varchar(512);not null"`
	ThumbnailURL   string `json:"thumbnail_url,omitempty" gorm:"type:varchar(512)"`
	OrganizationID string `json:"organization_id" gorm:"type:varchar(36);index;not null"`
	CameraID       string `json:"camera_id" gorm:"type:varchar(36);index;not null"`

	// Relationships
	Camera       Camera       `json:"camera,omitempty" gorm:"foreignKey:CameraID"`
	Organization Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	Person       Person       `json:"person,omitempty" gorm:"foreignKey:PersonHash;references:PersonHash"`
}

// TableName specifies the table name for FaceImage
func (FaceImage) TableName() string {
	return "face_images"
}