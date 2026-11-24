package domain

import "time"

type Site struct {
	ID             int64          `json:"id"`
	Key            string         `json:"key"`
	Name           string         `json:"name"`
	DisplayName    string         `json:"displayName"`
	Status         SiteStatus     `json:"status"`
	ModerationMode ModerationMode `json:"moderationMode"`
	ThemeColor     string         `json:"themeColor"`
	DefaultLocale  string         `json:"defaultLocale"`
	Origins        []string       `json:"origins"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type User struct {
	ID           int64      `json:"id"`
	DisplayName  string     `json:"displayName"`
	AvatarURL    string     `json:"avatarUrl"`
	Status       UserStatus `json:"status"`
	IsSuperadmin bool       `json:"isSuperadmin"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// Author 是批注/评论里展示的最小用户信息。
type Author struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
}

type Identity struct {
	ID            int64        `json:"id"`
	UserID        int64        `json:"userId"`
	Provider      ProviderType `json:"provider"`
	SiteID        int64        `json:"siteId"` // 0 = 全局（GitHub）
	Issuer        string       `json:"issuer"`
	Subject       string       `json:"subject"`
	Handle        string       `json:"handle"`
	Email         string       `json:"email,omitempty"`
	EmailVerified bool         `json:"emailVerified"`
	AvatarURL     string       `json:"avatarUrl"`
	CreatedAt     time.Time    `json:"createdAt"`
	LastLoginAt   time.Time    `json:"lastLoginAt"`
}

type Session struct {
	ID         int64      `json:"id"`
	TokenHash  string     `json:"-"`
	UserID     int64      `json:"-"`
	UserAgent  string     `json:"userAgent"`
	IP         string     `json:"ip"`
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type QuoteSelector struct {
	Type   string `json:"type"`
	Exact  string `json:"exact"`
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

type TextPosition struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Anchor struct {
	Quote    QuoteSelector `json:"quote"`
	Position *TextPosition `json:"position,omitempty"`
	Section  string        `json:"section,omitempty"`
}

type Annotation struct {
	ID         int64            `json:"id"`
	SiteID     int64            `json:"-"`
	PageKey    string           `json:"pageKey"`
	PageTitle  string           `json:"pageTitle"`
	Anchor     Anchor           `json:"anchor"`
	QuoteExact string           `json:"quoteExact"`
	Section    string           `json:"section"`
	Status     AnnotationStatus `json:"status"`
	Moderation ModerationStatus `json:"moderation"`
	Visibility Visibility       `json:"visibility"`
	CreatedBy  int64            `json:"-"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
	Author     Author           `json:"author"`
	Comments   []Comment        `json:"comments"`
}

type Comment struct {
	ID           int64            `json:"id"`
	SiteID       int64            `json:"-"`
	AnnotationID int64            `json:"annotationId"`
	UserID       int64            `json:"-"`
	Content      string           `json:"content"`
	Moderation   ModerationStatus `json:"moderation"`
	Visibility   Visibility       `json:"visibility"`
	CreatedAt    time.Time        `json:"createdAt"`
	Author       Author           `json:"author"`
}

type OIDCProvider struct {
	ID                  int64     `json:"id"`
	SiteID              int64     `json:"siteId"`
	Name                string    `json:"name"`
	Issuer              string    `json:"issuer"`
	ClientID            string    `json:"clientId"`
	ClientSecretCipher  []byte    `json:"-"`
	Scopes              []string  `json:"scopes"`
	AllowedEmailDomains []string  `json:"allowedEmailDomains"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type SiteAdmin struct {
	SiteID    int64     `json:"siteId"`
	UserID    int64     `json:"userId"`
	GrantedBy int64     `json:"grantedBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type SiteBan struct {
	SiteID    int64     `json:"siteId"`
	UserID    int64     `json:"userId"`
	BannedBy  int64     `json:"bannedBy"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthFlow struct {
	StateHash    string
	Mode         FlowMode
	ProviderType ProviderType
	ProviderID   int64 // 0 = GitHub
	Issuer       string
	SiteID       int64
	UserID       int64 // bind 时为目标用户
	ReturnTo     string
	CodeVerifier string
	Nonce        string
	ExpiresAt    time.Time
}

// Principal 是一次请求鉴权后的主体。
type Principal struct {
	User         User
	IsSuperadmin bool
	SiteID       int64
	IsSiteAdmin  bool
}

func (p Principal) CanModerateSite(siteID int64) bool {
	return p.IsSuperadmin || (p.SiteID == siteID && p.IsSiteAdmin)
}
