package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 添加安全相关的 HTTP 响应头
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止 MIME 类型嗅探攻击
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止点击劫持攻击
		c.Header("X-Frame-Options", "DENY")

		// XSS 防护（现代浏览器已内置，但保留以兼容旧浏览器）
		c.Header("X-XSS-Protection", "1; mode=block")

		// 内容安全策略（CSP）- 仅允许同源资源
		// 注意：如果前端需要加载外部资源（CDN、第三方脚本等），需要调整此策略
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		// 引用来源策略 - 控制 Referer 头的发送
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 权限策略 - 禁用不需要的浏览器特性
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")

		// HSTS（HTTP 严格传输安全）- 仅在 HTTPS 环境下启用
		// 强制浏览器使用 HTTPS 访问，防止协议降级攻击
		if c.Request.TLS != nil {
			// max-age=31536000: 1年有效期
			// includeSubDomains: 包含所有子域名
			// preload: 允许加入 HSTS 预加载列表
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}
