package models

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Message{},
		&RefreshToken{},
		&User{},
		&AdvertiserRequest{},
		&AdPricePlan{},
		&Event{},
		&EventParticipant{},
		&EventReview{},
		&EmailVerification{},
		&RegistrationPending{},
		&PasswordReset{},
		&Interest{},
		&UserInterest{},
		&EventMatching{},
		&MatchRequest{},
		&MicroCommunity{},
		&CommunityMember{},
		&CommunityInterest{},
		&Category{},
		&EventCategory{},
		&Subscription{},
		&Payment{},
		&PromotionPackage{},
		&EventPromotion{},
		&AdBanner{},
	)
}

func IsValidUserRole(role UserRole) bool {
	return role == RoleUser || role == RoleAdmin || role == RoleAdvertiser
}

func IsValidUserStatus(status UserStatus) bool {
	return status == UserStatusActive || status == UserStatusDeleted
}

func IsValidEventStatus(status EventStatus) bool {
	return status == EventStatusActive || status == EventStatusPast || status == EventStatusRejected
}

