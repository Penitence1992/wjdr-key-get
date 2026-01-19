package service

import (
	"context"
	"fmt"
	"time"

	"cdk-get/internal/notification"
	"cdk-get/internal/storage"

	"github.com/sirupsen/logrus"
)

// NotificationService handles notification sending and persistence
type NotificationService struct {
	notifier   notification.Notifier
	repository storage.Repository
	logger     *logrus.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notifier notification.Notifier,
	repository storage.Repository,
	logger *logrus.Logger,
) *NotificationService {
	return &NotificationService{
		notifier:   notifier,
		repository: repository,
		logger:     logger,
	}
}

// SendAndSave sends a notification and saves the result to database
func (s *NotificationService) SendAndSave(ctx context.Context, title, summary, content string) error {
	// Send notification
	req := notification.NotificationRequest{
		Title:   title,
		Summary: summary,
		Content: content,
	}

	s.logger.WithFields(logrus.Fields{
		"title":   title,
		"summary": summary,
		"channel": s.notifier.GetChannel(),
	}).Info("sending notification")

	result, err := s.notifier.Send(ctx, req)

	// Prepare notification record
	notif := &storage.Notification{
		Channel:   s.notifier.GetChannel(),
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Set status and result based on send outcome
	if err != nil || (result != nil && !result.Success) {
		notif.Status = storage.NotificationStatusFailed
		if err != nil {
			notif.Result = fmt.Sprintf("Error: %v", err)
		} else if result != nil && result.Message != "" {
			notif.Result = result.Message
		} else {
			notif.Result = "Unknown error"
		}

		s.logger.WithFields(logrus.Fields{
			"title":  title,
			"error":  err,
			"result": notif.Result,
		}).Error("notification send failed")
	} else {
		notif.Status = storage.NotificationStatusSuccess
		if result != nil && result.Message != "" {
			notif.Result = result.Message
		} else {
			notif.Result = "Notification sent successfully"
		}

		s.logger.WithFields(logrus.Fields{
			"title":  title,
			"result": notif.Result,
		}).Info("notification sent successfully")
	}

	// Save to database
	if saveErr := s.repository.SaveNotification(ctx, notif); saveErr != nil {
		s.logger.WithFields(logrus.Fields{
			"title": title,
			"error": saveErr.Error(),
		}).Error("failed to save notification record")
		return saveErr
	}

	s.logger.WithFields(logrus.Fields{
		"title":      title,
		"channel":    notif.Channel,
		"status":     notif.Status,
		"created_at": notif.CreatedAt,
	}).Info("notification record saved to database")

	return err
}
