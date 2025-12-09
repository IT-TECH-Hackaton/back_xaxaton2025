package services

import (
	"strings"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

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
	query := database.DB.Preload("Members").Where("auto_notify = ?", true)

	if err := query.Find(&communities).Error; err != nil {
		cs.logger.Error("Ошибка получения сообществ для уведомлений", zap.Error(err))
		return
	}

	eventTags := strings.ToLower(event.FullDescription + " " + event.ShortDescription + " " + event.Title)

	for _, community := range communities {
		if !community.AutoNotify {
			continue
		}

		communityTags := strings.ToLower(community.InterestTags)
		tags := strings.Split(communityTags, ",")
		matched := false

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" && strings.Contains(eventTags, tag) {
				matched = true
				break
			}
		}

		if matched {
			cs.notifyCommunityMembers(community, event)
		}
	}
}

func (cs *CommunityService) notifyCommunityMembers(community models.MicroCommunity, event *models.Event) {
	emailService := NewEmailService()

	for _, member := range community.Members {
		go func(m models.CommunityMember, e *models.Event, comm models.MicroCommunity) {
			if err := emailService.SendCommunityEventNotification(m.User.Email, m.User.FullName, comm.Name, e); err != nil {
				cs.logger.Error("Ошибка отправки уведомления сообществу",
					zap.String("email", m.User.Email),
					zap.String("community", comm.Name),
					zap.Error(err),
				)
			}
		}(member, event, community)
	}
}

