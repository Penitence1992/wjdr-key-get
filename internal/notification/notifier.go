package notification

import (
	"context"
)

// Notifier defines the interface for sending notifications
type Notifier interface {
	// Send sends a notification and returns the result
	Send(ctx context.Context, req NotificationRequest) (*NotificationResult, error)

	// GetChannel returns the channel name for this notifier
	GetChannel() string
}

// NotificationRequest contains notification parameters
type NotificationRequest struct {
	Title   string
	Summary string
	Content string
}

// NotificationResult contains the notification send result
type NotificationResult struct {
	Success bool
	Message string
	Error   error
}
