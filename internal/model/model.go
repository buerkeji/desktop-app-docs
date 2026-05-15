package model

type DesktopBootstrap struct {
	Sites       []SiteItem        `json:"sites"`
	Tenants     []TenantItem      `json:"tenants"`
	Token       *TenantTokenState `json:"token,omitempty"`
	User        *CurrentUser      `json:"user,omitempty"`
	TenantAuths []TenantAuthEntry `json:"tenantAuths"`
}

type TenantAuthEntry struct {
	TenantID int64            `json:"tenantId"`
	Token    TenantTokenState `json:"token"`
	User     CurrentUser      `json:"user"`
}

type SiteItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	BaseURL     string `json:"baseUrl"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"isDefault,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type TenantCapabilities struct {
	Article    bool `json:"article"`
	Tool       bool `json:"tool"`
	Dictionary bool `json:"dictionary"`
	Media      bool `json:"media"`
}

type TenantLimits struct {
	MaxUploadMB   int `json:"maxUploadMb"`
	MaxBatchCount int `json:"maxBatchCount"`
}

type TenantItem struct {
	ID           int64              `json:"id"`
	SiteID       int64              `json:"siteId"`
	Name         string             `json:"name"`
	BaseURL      string             `json:"baseUrl"`
	APIBaseURL   string             `json:"apiBaseUrl"`
	TenantName   string             `json:"tenantName,omitempty"`
	TenantSlug   string             `json:"tenantSlug,omitempty"`
	LastUsername string             `json:"lastUsername,omitempty"`
	Status       string             `json:"status"`
	Capabilities TenantCapabilities `json:"capabilities"`
	Limits       TenantLimits       `json:"limits"`
	CreatedAt    string             `json:"createdAt"`
}

type CurrentUser struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

type TenantTokenState struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	TokenType        string `json:"tokenType"`
	ExpiresAt        string `json:"expiresAt"`
	RefreshExpiresAt string `json:"refreshExpiresAt"`
	SessionID        int64  `json:"sessionId"`
	TenantID         int64  `json:"tenantId"`
}

type AuthBootstrap struct {
	Token TenantTokenState `json:"token"`
	User  CurrentUser      `json:"user"`
}

type CreateSiteInput struct {
	Name        string `json:"name"`
	BaseURL     string `json:"baseUrl"`
	Description string `json:"description"`
}

type UpdateSiteInput struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	BaseURL     string `json:"baseUrl"`
	Description string `json:"description"`
}

type DeleteSiteInput struct {
	ID int64 `json:"id"`
}

type CreateTenantInput struct {
	SiteID       int64  `json:"siteId"`
	Name         string `json:"name"`
	BaseURL      string `json:"baseUrl"`
	APIBaseURL   string `json:"apiBaseUrl"`
	TenantName   string `json:"tenantName"`
	TenantSlug   string `json:"tenantSlug"`
	LastUsername string `json:"lastUsername"`
}

type UpdateTenantInput struct {
	ID           int64  `json:"id"`
	SiteID       int64  `json:"siteId"`
	Name         string `json:"name"`
	BaseURL      string `json:"baseUrl"`
	APIBaseURL   string `json:"apiBaseUrl"`
	TenantName   string `json:"tenantName"`
	TenantSlug   string `json:"tenantSlug"`
	LastUsername string `json:"lastUsername"`
}

type DeleteTenantInput struct {
	ID int64 `json:"id"`
}

type DiscoverSiteTenantsInput struct {
	BaseURL string `json:"baseUrl"`
}

type SyncSiteTenantsInput struct {
	SiteID int64 `json:"siteId"`
}

type TenantDomainItem struct {
	ID        int64  `json:"id"`
	Domain    string `json:"domain"`
	IsPrimary bool   `json:"isPrimary"`
	IsActive  bool   `json:"isActive"`
}

type TenantDiscoveryLoginHint struct {
	Mode       string `json:"mode"`
	TenantSlug string `json:"tenantSlug"`
}

type TenantDiscoveryItem struct {
	Slug               string                   `json:"slug"`
	Name               string                   `json:"name"`
	HasActiveDomain    bool                     `json:"hasActiveDomain"`
	RecommendedBaseURL string                   `json:"recommendedBaseUrl"`
	APIBaseURL         string                   `json:"apiBaseUrl"`
	LoginHint          TenantDiscoveryLoginHint `json:"loginHint"`
}

type SiteDiscoveryInfo struct {
	Host         string `json:"host"`
	BaseURL      string `json:"baseUrl"`
	IsSystemHost bool   `json:"isSystemHost"`
}

type TenantDiscoveryPayload struct {
	Site    SiteDiscoveryInfo     `json:"site"`
	Tenants []TenantDiscoveryItem `json:"tenants"`
}

type SaveTenantAuthInput struct {
	TenantID         int64    `json:"tenantId"`
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	TokenType        string   `json:"tokenType"`
	ExpiresAt        string   `json:"expiresAt"`
	RefreshExpiresAt string   `json:"refreshExpiresAt"`
	SessionID        int64    `json:"sessionId"`
	UserID           int64    `json:"userId"`
	Username         string   `json:"username"`
	Name             string   `json:"name"`
	Roles            []string `json:"roles"`
}

type LoginTenantInput struct {
	TenantID      int64  `json:"tenantId"`
	APIBaseURL    string `json:"apiBaseUrl"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	TenantSlug    string `json:"tenantSlug"`
	DeviceName    string `json:"deviceName"`
	ClientVersion string `json:"clientVersion"`
}

type RefreshTenantTokenInput struct {
	TenantID      int64  `json:"tenantId"`
	APIBaseURL    string `json:"apiBaseUrl"`
	RefreshToken  string `json:"refreshToken"`
	DeviceName    string `json:"deviceName"`
	ClientVersion string `json:"clientVersion"`
}

type LogoutTenantInput struct {
	APIBaseURL  string `json:"apiBaseUrl"`
	AccessToken string `json:"accessToken"`
}

type LocalDraftQueryInput struct {
	TenantID    int64  `json:"tenantId"`
	ContentType string `json:"contentType"`
	TargetID    string `json:"targetId"`
}

type LocalDraftListInput struct {
	TenantID    int64  `json:"tenantId"`
	ContentType string `json:"contentType,omitempty"`
}

type SaveLocalDraftInput struct {
	TenantID    int64  `json:"tenantId"`
	ContentType string `json:"contentType"`
	TargetID    string `json:"targetId"`
	Title       string `json:"title"`
	PayloadJSON string `json:"payloadJson"`
}

type LocalDraftItem struct {
	ID          int64  `json:"id"`
	TenantID    int64  `json:"tenantId"`
	ContentType string `json:"contentType"`
	TargetID    string `json:"targetId"`
	Title       string `json:"title,omitempty"`
	PayloadJSON string `json:"payloadJson"`
	UpdatedAt   string `json:"updatedAt"`
}

type RemoteDesktopAPIRequestInput struct {
	APIBaseURL  string            `json:"apiBaseUrl"`
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	AccessToken string            `json:"accessToken"`
}

type RemoteDesktopAPIResponse struct {
	Success    bool                `json:"success"`
	Message    string              `json:"message"`
	Data       any                 `json:"data"`
	Errors     map[string][]string `json:"errors,omitempty"`
	ErrorCode  string              `json:"error_code,omitempty"`
	RequestID  string              `json:"request_id,omitempty"`
	StatusCode int                 `json:"status_code,omitempty"`
}

type UploadDesktopMediaInput struct {
	APIBaseURL      string `json:"apiBaseUrl"`
	AccessToken     string `json:"accessToken"`
	FileName        string `json:"fileName"`
	MimeType        string `json:"mimeType"`
	FileBase64      string `json:"fileBase64"`
	OriginalName    string `json:"originalName"`
	MediaCategoryID int64  `json:"mediaCategoryId"`
	SourceURL       string `json:"sourceUrl"`
	DraftID         int64  `json:"draftId"`
	UploadScene     string `json:"uploadScene"`
}

type DownloadRemoteMediaInput struct {
	URL     string `json:"url"`
	Referer string `json:"referer"`
}

type DownloadRemoteMediaResult struct {
	FileBase64 string `json:"fileBase64"`
	MimeType   string `json:"mimeType"`
	FileName   string `json:"fileName"`
	FileSize   int64  `json:"fileSize"`
}

type TenantScopeInfo struct {
	SiteID     int64
	TenantID   int64
	TenantName string
}

type SubmitRecordInput struct {
	SiteID         int64  `json:"siteId"`
	TenantID       int64  `json:"tenantId"`
	Title          string `json:"title"`
	ContentType    string `json:"contentType"`
	JobType        string `json:"jobType"`
	IdempotencyKey string `json:"idempotencyKey"`
	PayloadJSON    string `json:"payloadJson"`
	Status         string `json:"status"`
	StartedAt      string `json:"startedAt"`
	FinishedAt     string `json:"finishedAt"`
	DraftID        int64  `json:"draftId"`
	RemoteID       int64  `json:"remoteId"`
	RemoteURL      string `json:"remoteUrl"`
	MatchType      string `json:"matchType"`
	CreatedCount   int    `json:"createdCount"`
	UpdatedCount   int    `json:"updatedCount"`
	FailedCount    int    `json:"failedCount"`
	ResultJSON     string `json:"resultJson"`
	ErrorMessage   string `json:"errorMessage"`
}

type SubmitRecordListInput struct {
	TenantID    int64  `json:"tenantId"`
	Keyword     string `json:"keyword"`
	ContentType string `json:"contentType"`
	Status      string `json:"status"`
	DateFrom    string `json:"dateFrom"`
	DateTo      string `json:"dateTo"`
	Limit       int    `json:"limit"`
}

type SubmitRecordItem struct {
	ID             int64  `json:"id"`
	JobID          int64  `json:"jobId"`
	SiteID         int64  `json:"siteId"`
	TenantID       int64  `json:"tenantId"`
	TenantName     string `json:"tenantName"`
	Title          string `json:"title"`
	ContentType    string `json:"contentType"`
	JobType        string `json:"jobType"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotencyKey"`
	RemoteID       int64  `json:"remoteId"`
	RemoteURL      string `json:"remoteUrl"`
	MatchType      string `json:"matchType"`
	CreatedCount   int    `json:"createdCount"`
	UpdatedCount   int    `json:"updatedCount"`
	FailedCount    int    `json:"failedCount"`
	ErrorMessage   string `json:"errorMessage"`
	PayloadJSON    string `json:"payloadJson"`
	ResultJSON     string `json:"resultJson"`
	SubmittedAt    string `json:"submittedAt"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type AppLogInput struct {
	SiteID      int64  `json:"siteId"`
	TenantID    int64  `json:"tenantId"`
	RequestID   string `json:"requestId"`
	Level       string `json:"level"`
	Module      string `json:"module"`
	Message     string `json:"message"`
	ContextJSON string `json:"contextJson"`
}

type AppLogListInput struct {
	TenantID  int64  `json:"tenantId"`
	Keyword   string `json:"keyword"`
	Level     string `json:"level"`
	Module    string `json:"module"`
	RequestID string `json:"requestId"`
	DateFrom  string `json:"dateFrom"`
	DateTo    string `json:"dateTo"`
	Limit     int    `json:"limit"`
}

type AppLogItem struct {
	ID          int64  `json:"id"`
	SiteID      int64  `json:"siteId"`
	TenantID    int64  `json:"tenantId"`
	TenantName  string `json:"tenantName"`
	RequestID   string `json:"requestId"`
	Level       string `json:"level"`
	Module      string `json:"module"`
	Message     string `json:"message"`
	ContextJSON string `json:"contextJson"`
	CreatedAt   string `json:"createdAt"`
}

type MediaTaskInput struct {
	SiteID          int64  `json:"siteId"`
	TenantID        int64  `json:"tenantId"`
	FileName        string `json:"fileName"`
	OriginalName    string `json:"originalName"`
	MimeType        string `json:"mimeType"`
	UploadScene     string `json:"uploadScene"`
	CachedFilePath  string `json:"cachedFilePath"`
	SourceURL       string `json:"sourceUrl"`
	MediaCategoryID int64  `json:"mediaCategoryId"`
	DraftID         int64  `json:"draftId"`
	Status          string `json:"status"`
	RequestID       string `json:"requestId"`
	RemoteMediaID   int64  `json:"remoteMediaId"`
	RemoteURL       string `json:"remoteUrl"`
	RemotePath      string `json:"remotePath"`
	Disk            string `json:"disk"`
	SizeBytes       int64  `json:"sizeBytes"`
	Width           int64  `json:"width"`
	Height          int64  `json:"height"`
	ErrorMessage    string `json:"errorMessage"`
	ResponseJSON    string `json:"responseJson"`
}

type MediaTaskListInput struct {
	TenantID int64  `json:"tenantId"`
	Keyword  string `json:"keyword"`
	Scene    string `json:"scene"`
	Status   string `json:"status"`
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
	Limit    int    `json:"limit"`
}

type MediaTaskItem struct {
	ID              int64  `json:"id"`
	SiteID          int64  `json:"siteId"`
	TenantID        int64  `json:"tenantId"`
	TenantName      string `json:"tenantName"`
	FileName        string `json:"fileName"`
	OriginalName    string `json:"originalName"`
	MimeType        string `json:"mimeType"`
	UploadScene     string `json:"uploadScene"`
	CanRetry        bool   `json:"canRetry"`
	SourceURL       string `json:"sourceUrl"`
	MediaCategoryID int64  `json:"mediaCategoryId"`
	DraftID         int64  `json:"draftId"`
	Status          string `json:"status"`
	RequestID       string `json:"requestId"`
	RemoteMediaID   int64  `json:"remoteMediaId"`
	RemoteURL       string `json:"remoteUrl"`
	RemotePath      string `json:"remotePath"`
	Disk            string `json:"disk"`
	SizeBytes       int64  `json:"sizeBytes"`
	Width           int64  `json:"width"`
	Height          int64  `json:"height"`
	ErrorMessage    string `json:"errorMessage"`
	ResponseJSON    string `json:"responseJson"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type MediaTaskRetryInput struct {
	TaskID int64 `json:"taskId"`
}

type MediaTaskCacheCleanupInput struct {
	TaskID int64 `json:"taskId"`
}

type MediaTaskRetryInfo struct {
	TaskID          int64
	SiteID          int64
	TenantID        int64
	APIBaseURL      string
	AccessToken     string
	FileName        string
	OriginalName    string
	MimeType        string
	UploadScene     string
	CachedFilePath  string
	SourceURL       string
	MediaCategoryID int64
	DraftID         int64
}

type MediaTaskCacheInfo struct {
	TaskID         int64
	CachedFilePath string
}

type CollectWebPageInput struct {
	URL              string `json:"url"`
	SiteID           int64  `json:"siteId,omitempty"`
	TenantID         int64  `json:"tenantId,omitempty"`
	UseBrowserRender bool   `json:"useBrowserRender,omitempty"`
}

type CollectedImage struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

type CollectedWebPageResult struct {
	RequestedURL       string           `json:"requestedUrl"`
	FinalURL           string           `json:"finalUrl"`
	CanonicalURL       string           `json:"canonicalUrl,omitempty"`
	OfficialURL        string           `json:"officialUrl,omitempty"`
	Host               string           `json:"host"`
	SourceHTML         string           `json:"sourceHtml,omitempty"`
	BrowserPreviewHTML string           `json:"browserPreviewHtml,omitempty"`
	SiteName           string           `json:"siteName,omitempty"`
	Title              string           `json:"title"`
	Description        string           `json:"description,omitempty"`
	Excerpt            string           `json:"excerpt,omitempty"`
	IconURL            string           `json:"iconUrl,omitempty"`
	ThumbnailURL       string           `json:"thumbnailUrl,omitempty"`
	ContentHTML        string           `json:"contentHtml"`
	ContentText        string           `json:"contentText"`
	Keywords           []string         `json:"keywords"`
	SeoTitle           string           `json:"seoTitle,omitempty"`
	SeoDescription     string           `json:"seoDescription,omitempty"`
	SeoKeywords        []string         `json:"seoKeywords"`
	SuggestedTags      []string         `json:"suggestedTags"`
	Images             []CollectedImage `json:"images"`
	PublishedAt        string           `json:"publishedAt,omitempty"`
	FetchedAt          string           `json:"fetchedAt"`
}

type BrowserRenderedPreviewResult struct {
	RequestedURL string `json:"requestedUrl"`
	FinalURL     string `json:"finalUrl"`
	Host         string `json:"host"`
	Browser      string `json:"browser"`
	HTML         string `json:"html"`
	RenderedAt   string `json:"renderedAt"`
}

type ScanSiteLinksInput struct {
	URL         string           `json:"url"`
	MaxLinks    int              `json:"maxLinks"`
	ScanSitemap bool             `json:"scanSitemap"`
	FilterRules []ScanFilterRule `json:"filterRules,omitempty"`
	SiteID      int64            `json:"siteId,omitempty"`
	TenantID    int64            `json:"tenantId,omitempty"`
}

type ScanFilterRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type SiteLinkItem struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Source string `json:"source,omitempty"`
}

type ScanSiteLinksResult struct {
	RequestedURL    string         `json:"requestedUrl"`
	FinalURL        string         `json:"finalUrl"`
	Host            string         `json:"host"`
	SiteName        string         `json:"siteName,omitempty"`
	Title           string         `json:"title"`
	Links           []SiteLinkItem `json:"links"`
	PageHTMLCount   int            `json:"pageHtmlCount"`
	SitemapURLCount int            `json:"sitemapUrlCount"`
	SitemapSources  []string       `json:"sitemapSources,omitempty"`
	ScannedAt       string         `json:"scannedAt"`
}
