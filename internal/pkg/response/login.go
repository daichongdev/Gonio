package response

// LoginResult 登录返回结果
type LoginResult struct {
	Token    string `json:"token"`
	ExpireAt int64  `json:"expire_at"`
	User     any    `json:"user"`
}

// UserInfo 用户信息（用于登录响应，替代 gin.H 使 service 层与框架解耦）
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// AdminInfo 管理员信息（用于登录响应）
type AdminInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
}
