package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials 凭证无效错误
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken 令牌无效错误
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired 令牌过期错误
	ErrTokenExpired = errors.New("token expired")
)

// Claims JWT声明结构
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthService 认证服务接口
type AuthService interface {
	// ValidateCredentials 验证管理员凭证
	ValidateCredentials(username, password string) error

	// GenerateToken 生成JWT令牌
	GenerateToken(username string) (token string, expiresAt time.Time, err error)

	// ValidateToken 验证JWT令牌
	ValidateToken(token string) (*Claims, error)
}

// authServiceImpl 认证服务实现
type authServiceImpl struct {
	username      string
	passwordHash  string
	tokenSecret   []byte
	tokenDuration time.Duration
}

// NewAuthService 创建认证服务实例
func NewAuthService(username, passwordHash, tokenSecret string, tokenDuration time.Duration) AuthService {
	return &authServiceImpl{
		username:      username,
		passwordHash:  passwordHash,
		tokenSecret:   []byte(tokenSecret),
		tokenDuration: tokenDuration,
	}
}

// ValidateCredentials 验证管理员凭证
func (s *authServiceImpl) ValidateCredentials(username, password string) error {
	// 验证用户名
	if username != s.username {
		return ErrInvalidCredentials
	}

	// 使用bcrypt验证密码
	err := bcrypt.CompareHashAndPassword([]byte(s.passwordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("password verification failed: %w", err)
	}

	return nil
}

// GenerateToken 生成JWT令牌
func (s *authServiceImpl) GenerateToken(username string) (string, time.Time, error) {
	// 计算过期时间
	expiresAt := time.Now().Add(s.tokenDuration)

	// 创建声明
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "admin-dashboard",
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString(s.tokenSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken 验证JWT令牌
func (s *authServiceImpl) ValidateToken(tokenString string) (*Claims, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.tokenSecret, nil
	})

	if err != nil {
		// 检查是否是过期错误
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// 提取声明
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
