package dto

type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required"`
	Comment string `json:"comment"`
}

type UpdateReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type ReviewResponse struct {
	ID        string    `json:"id"`
	EventID   string    `json:"eventID"`
	UserID    string    `json:"userID"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	User      UserInfo  `json:"user"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

type ReviewsResponse struct {
	Data          []ReviewResponse `json:"data"`
	AverageRating float64          `json:"averageRating"`
	TotalReviews  int64            `json:"totalReviews"`
	Pagination    Pagination       `json:"pagination"`
}

