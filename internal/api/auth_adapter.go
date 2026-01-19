package api

import (
	"cdk-get/internal/auth"
)

// authServiceAdapter 适配器，将auth.AuthService转换为middleware.AuthService
type authServiceAdapter struct {
	authService auth.AuthService
}

// NewAuthServiceAdapter 创建认证服务适配器
func NewAuthServiceAdapter(authService auth.AuthService) AuthService {
	return &authServiceAdapter{
		authService: authService,
	}
}

// ValidateToken 验证JWT令牌并返回声明
func (a *authServiceAdapter) ValidateToken(token string) (*AuthClaims, error) {
	claims, err := a.authService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	return &AuthClaims{
		Username: claims.Username,
	}, nil
}
