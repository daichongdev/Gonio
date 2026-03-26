package req

// LoginReq 登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required" label:"用户名"`
	Password string `json:"password" binding:"required,min=6,max=72" label:"密码"`
}
