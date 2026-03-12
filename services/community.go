package services

import (
	"strings"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CommunityService struct {
	logger *zap.Logger
}

func NewCommunityService() *CommunityService {
	return &CommunityService{
		logger: utils.GetLogger(),
	}
}

func (cs *CommunityService) NotifyCommunitiesAboutEvent(event *models.Event) {
	var communities []models.MicroCommunity
	query := database.DB.Preload("Members.User").Preload("Interests").Where("auto_notify = ?", true)

	if err := query.Find(&communities).Error; err != nil {
		cs.logger.Error("Ошибка получения сообществ для уведомлений", zap.Error(err))
		return
	}

	for _, community := range communities {
		if !community.AutoNotify {
			continue
		}

		if len(community.Interests) == 0 {
			continue
		}

		matched := cs.checkEventMatchesCommunityInterests(event, community)
		if matched {
			cs.notifyCommunityMembers(community, event)
		}
	}
}

func (cs *CommunityService) checkEventMatchesCommunityInterests(event *models.Event, community models.MicroCommunity) bool {
	if len(community.Interests) == 0 {
		return false
	}

	communityInterestIDs := make(map[uuid.UUID]bool)
	communityInterestCategories := make(map[string]bool)
	communityInterestNames := make(map[string]bool)

	for _, interest := range community.Interests {
		communityInterestIDs[interest.ID] = true
		if interest.Category != "" {
			communityInterestCategories[interest.Category] = true
		}
		communityInterestNames[interest.Name] = true
	}

	if err := database.DB.Preload("Categories").Where("id = ?", event.ID).First(event).Error; err != nil {
		cs.logger.Error("Ошибка загрузки категорий события", zap.Error(err))
	}

	for _, category := range event.Categories {
		if communityInterestCategories[category.Name] {
			return true
		}
	}

	for _, tag := range event.Tags {
		tagLower := strings.ToLower(strings.TrimSpace(tag))
		for interestName := range communityInterestNames {
			if strings.Contains(strings.ToLower(interestName), tagLower) || strings.Contains(tagLower, strings.ToLower(interestName)) {
				return true
			}
		}
	}

	var eventParticipants []models.EventParticipant
	if err := database.DB.Preload("User").Where("event_id = ?", event.ID).Find(&eventParticipants).Error; err != nil {
		cs.logger.Error("Ошибка получения участников события", zap.Error(err))
		return false
	}

	if len(eventParticipants) == 0 {
		return false
	}

	for _, participant := range eventParticipants {
		var userInterests []models.UserInterest
		if err := database.DB.Preload("Interest").Where("user_id = ?", participant.UserID).Find(&userInterests).Error; err != nil {
			continue
		}

		for _, userInterest := range userInterests {
			if communityInterestIDs[userInterest.InterestID] {
				if userInterest.Weight >= 5 {
					return true
				}
			}
		}
	}

	return false
}

func (cs *CommunityService) notifyCommunityMembers(community models.MicroCommunity, event *models.Event) {
	emailService := NewEmailService()

	for _, member := range community.Members {
		if member.User.ID == uuid.Nil {
			continue
		}

		go func(m models.CommunityMember, e *models.Event, comm models.MicroCommunity) {
			var user models.User
			if err := database.DB.Where("id = ?", m.UserID).First(&user).Error; err != nil {
				cs.logger.Error("Ошибка получения пользователя для уведомления",
					zap.String("userID", m.UserID.String()),
					zap.Error(err),
				)
				return
			}

			if err := emailService.SendCommunityEventNotification(user.Email, user.FullName, comm.Name, e); err != nil {
				cs.logger.Error("Ошибка отправки уведомления сообществу",
					zap.String("email", user.Email),
					zap.String("community", comm.Name),
					zap.Error(err),
				)
			}
		}(member, event, community)
	}
}
