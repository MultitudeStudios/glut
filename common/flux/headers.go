package flux

// Headers
const (
	HeaderAuthorization        = "Authorization"
	HeaderContentEncoding      = "Content-Encoding"
	HeaderContentLength        = "Content-Length"
	HeaderContentType          = "Content-Type"
	ContentTypeApplicationJSON = "application/json; charset=UTF-8"
	HeaderXForwardedFor        = "X-Forwarded-For"
	HeaderXRealIP              = "X-Real-Ip"
	HeaderXRequestID           = "X-Request-Id"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderReferrerPolicy          = "Referrer-Policy"
)
