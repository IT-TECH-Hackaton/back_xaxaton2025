package dto

type CreateCommunityRequest struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	InterestIDs []string  `json:"interestIDs"`
	AutoNotify  bool      `json:"autoNotify"`
}

type UpdateCommunityRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	InterestIDs  []string `json:"interestIDs"`
	AutoNotify  *bool    `json:"autoNotify"`
}

type CommunityResponse struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Interests    []InterestInfo  `json:"interests"`
	Admin        UserInfo        `json:"admin"`
	AutoNotify   bool            `json:"autoNotify"`
	MembersCount int             `json:"membersCount"`
	CreatedAt    string          `json:"createdAt"`
}

type InterestInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type CommunityMemberResponse struct {
	ID        string   `json:"id"`
	User      UserInfo `json:"user"`
	JoinedAt  string   `json:"joinedAt"`
}

