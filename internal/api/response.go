package api

// Response 标准响应格式
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(code, message string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}
