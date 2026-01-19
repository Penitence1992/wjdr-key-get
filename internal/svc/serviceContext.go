package svc

import (
	"cdk-get/internal/service"
	"cdk-get/internal/storage"
)

type ServiceContext struct {
	SqlClient           storage.KeyStorage
	Repository          storage.Repository
	NotificationService *service.NotificationService
}

func NewServiceContext(sqlClient storage.KeyStorage, repository storage.Repository, notificationService *service.NotificationService) *ServiceContext {
	return &ServiceContext{
		SqlClient:           sqlClient,
		Repository:          repository,
		NotificationService: notificationService,
	}
}
