package models

type UserRole string

const (
	UserRoleUser  UserRole = "USER"
	UserRoleAdmin UserRole = "ADMIN"
)

func (r UserRole) String() string {
	return string(r)
}

type UserStatus string

const (
	UserStatusActive  UserStatus = "ACTIVE"
	UserStatusDeleted UserStatus = "DELETED"
)

func (s UserStatus) String() string {
	return string(s)
}

type EventStatus string

const (
	EventStatusActive   EventStatus = "ACTIVE"
	EventStatusPast     EventStatus = "PAST"
	EventStatusRejected EventStatus = "REJECTED"
)

func (s EventStatus) String() string {
	return string(s)
}
