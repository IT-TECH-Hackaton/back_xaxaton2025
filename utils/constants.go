package utils

import "time"

const (
	MaxPageLimit              = 100
	DefaultPageLimit          = 20
	VerificationCodeExpiry    = 10 * time.Minute
	PasswordResetTokenExpiry  = 24 * time.Hour
	VerificationCodeLength    = 6
	MinPasswordLength         = 8
	MinFullNameLength         = 2
	MaxFullNameLength         = 100
	MaxTitleLength            = 200
	MaxShortDescriptionLength = 500
	MaxFullDescriptionLength  = 5000
	MaxAddressLength          = 500
	MaxMapLinkLength          = 1000
	MaxPaymentInfoLength      = 2000
	EventReminderHours        = 24
)

