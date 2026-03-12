package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MicroCommunity struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name            string    `gorm:"not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	AdminID         uuid.UUID `gorm:"type:uuid;not null" json:"adminID"`
	AutoNotify      bool      `gorm:"default:true" json:"autoNotify"`
	MembersCount    int       `gorm:"default:0" json:"membersCount"`
	Admin           User      `gorm:"foreignKey:AdminID" json:"admin"`
	Members         []CommunityMember `gorm:"foreignKey:CommunityID" json:"members"`
	Interests       []Interest `gorm:"many2many:community_interests;" json:"interests"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CommunityInterest struct {
	CommunityID uuid.UUID `gorm:"type:uuid;primary_key" json:"communityID"`
	InterestID  uuid.UUID `gorm:"type:uuid;primary_key" json:"interestID"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CommunityMember struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"userID"`
	CommunityID uuid.UUID `gorm:"type:uuid;not null;index" json:"communityID"`
	User        User      `gorm:"foreignKey:UserID" json:"user"`
	Community   MicroCommunity `gorm:"foreignKey:CommunityID" json:"community"`
	JoinedAt    time.Time `json:"joinedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (mc *MicroCommunity) BeforeCreate(tx *gorm.DB) error {
	if mc.ID == uuid.Nil {
		mc.ID = uuid.New()
	}
	return nil
}

func (cm *CommunityMember) BeforeCreate(tx *gorm.DB) error {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	return nil
}

