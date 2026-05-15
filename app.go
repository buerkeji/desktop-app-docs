package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"zq-desktop-app/internal/model"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	store   *DesktopStore
	initErr error
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	store, err := NewDesktopStore()
	if err != nil {
		a.initErr = err
		return
	}

	a.store = store

	runtime.WindowCenter(ctx)
}

func (a *App) CenterWindow() {
	if a.ctx != nil {
		runtime.WindowCenter(a.ctx)
	}
}

func (a *App) ensureStore() error {
	if a.initErr != nil {
		return a.initErr
	}

	if a.store == nil {
		return ErrStoreNotReady
	}

	return nil
}

func (a *App) GetDesktopBootstrap() (*model.DesktopBootstrap, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.GetDesktopBootstrap()
}

func (a *App) CreateSite(input model.CreateSiteInput) (*model.SiteItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.CreateSite(input)
}

func (a *App) UpdateSite(input model.UpdateSiteInput) (*model.SiteItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.UpdateSite(input)
}

func (a *App) DeleteSite(input model.DeleteSiteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteSite(input)
}

func (a *App) CreateTenant(input model.CreateTenantInput) (*model.TenantItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.CreateTenant(input)
}

func (a *App) UpdateTenant(input model.UpdateTenantInput) (*model.TenantItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.UpdateTenant(input)
}

func (a *App) DeleteTenant(input model.DeleteTenantInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteTenant(input)
}

func (a *App) DiscoverSiteTenants(input model.DiscoverSiteTenantsInput) (*model.TenantDiscoveryPayload, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return discoverSiteTenantsRemote(input)
}

func (a *App) SyncSiteTenants(input model.SyncSiteTenantsInput) ([]model.TenantItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	if input.SiteID <= 0 {
		return nil, fmt.Errorf("缺少站点 ID")
	}

	site, err := a.store.getSiteByID(input.SiteID)
	if err != nil {
		return nil, err
	}

	discovery, err := discoverSiteTenantsRemote(model.DiscoverSiteTenantsInput{
		BaseURL: site.BaseURL,
	})
	if err != nil {
		return nil, err
	}

	syncedTenants := make([]model.TenantItem, 0, len(discovery.Tenants))
	for _, item := range discovery.Tenants {
		tenant, err := a.store.CreateTenant(model.CreateTenantInput{
			SiteID:       site.ID,
			Name:         item.Name,
			BaseURL:      item.RecommendedBaseURL,
			APIBaseURL:   item.APIBaseURL,
			TenantName:   item.Name,
			TenantSlug:   item.Slug,
			LastUsername: "",
		})
		if err != nil {
			return nil, err
		}
		syncedTenants = append(syncedTenants, *tenant)
	}

	return syncedTenants, nil
}

func (a *App) LoginTenant(input model.LoginTenantInput) (*model.AuthBootstrap, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	if input.TenantID <= 0 {
		return nil, fmt.Errorf("缺少租户 ID")
	}

	result, err := loginTenantRemote(input)
	if err != nil {
		return nil, err
	}

	return a.store.SaveTenantAuth(model.SaveTenantAuthInput{
		TenantID:         input.TenantID,
		AccessToken:      result.AccessToken,
		RefreshToken:     result.RefreshToken,
		TokenType:        result.TokenType,
		ExpiresAt:        result.ExpiresAt,
		RefreshExpiresAt: result.RefreshExpiresAt,
		SessionID:        result.SessionID,
		UserID:           result.User.ID,
		Username:         result.User.Username,
		Name:             result.User.Name,
		Roles:            result.User.Roles,
	})
}

func (a *App) RefreshTenantToken(input model.RefreshTenantTokenInput) (*model.AuthBootstrap, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	if input.TenantID <= 0 {
		return nil, fmt.Errorf("缺少租户 ID")
	}

	result, err := refreshTenantTokenRemote(input)
	if err != nil {
		return nil, err
	}

	return a.store.SaveTenantAuth(model.SaveTenantAuthInput{
		TenantID:         input.TenantID,
		AccessToken:      result.AccessToken,
		RefreshToken:     result.RefreshToken,
		TokenType:        result.TokenType,
		ExpiresAt:        result.ExpiresAt,
		RefreshExpiresAt: result.RefreshExpiresAt,
		SessionID:        result.SessionID,
		UserID:           result.User.ID,
		Username:         result.User.Username,
		Name:             result.User.Name,
		Roles:            result.User.Roles,
	})
}

func (a *App) LogoutTenant(input model.LogoutTenantInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	if strings.TrimSpace(input.APIBaseURL) != "" && strings.TrimSpace(input.AccessToken) != "" {
		_ = logoutTenantRemote(input)
	}

	scope, _ := a.store.FindTenantScopeByAPIBaseURL(input.APIBaseURL)
	if scope != nil && scope.TenantID > 0 {
		return a.store.ClearTenantToken(scope.TenantID)
	}

	return a.store.ClearSavedTenantToken()
}

func (a *App) LogoutTenantById(tenantID int64) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.ClearTenantToken(tenantID)
}

func (a *App) ProxyDesktopApiRequest(input model.RemoteDesktopAPIRequestInput) (*model.RemoteDesktopAPIResponse, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	scope, _ := a.store.FindTenantScopeByAPIBaseURL(input.APIBaseURL)
	response, err := proxyDesktopAPIRequest(input)

	if logInput := buildAppLogInput(scope, input, response, err); logInput != nil {
		_ = a.store.RecordAppLog(*logInput)
	}
	if submitInput := buildSubmitRecordInput(scope, input, response, err); submitInput != nil {
		_ = a.store.RecordSubmitRecord(*submitInput)
	}

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (a *App) UploadDesktopMedia(input model.UploadDesktopMediaInput) (*model.RemoteDesktopAPIResponse, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	scope, _ := a.store.FindTenantScopeByAPIBaseURL(input.APIBaseURL)
	fileBytes, err := decodeDesktopUploadFileBase64(input.FileBase64)
	if err != nil {
		if mediaTask := buildMediaTaskInput(scope, input, "", nil, err); mediaTask != nil {
			_ = a.store.RecordMediaTask(*mediaTask)
		}
		return nil, err
	}

	cachedFilePath, cacheErr := a.store.SaveMediaUploadCache(input.FileName, fileBytes)
	if cacheErr != nil {
		cachedFilePath = ""
	}

	response, err := uploadDesktopMediaRemoteBytes(input, fileBytes)
	logRequest := model.RemoteDesktopAPIRequestInput{
		APIBaseURL: input.APIBaseURL,
		Method:     "POST",
		Path:       "/media/upload",
	}
	if logInput := buildAppLogInput(scope, logRequest, response, err); logInput != nil {
		_ = a.store.RecordAppLog(*logInput)
	}
	if mediaTask := buildMediaTaskInput(scope, input, cachedFilePath, response, err); mediaTask != nil {
		_ = a.store.RecordMediaTask(*mediaTask)
	}
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (a *App) RetryMediaTask(input model.MediaTaskRetryInput) (*model.RemoteDesktopAPIResponse, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	retryInfo, err := a.store.GetMediaTaskRetryInfo(input.TaskID)
	if err != nil {
		return nil, err
	}
	if retryInfo == nil {
		return nil, fmt.Errorf("媒体任务不存在")
	}
	if retryInfo.CachedFilePath == "" {
		return nil, fmt.Errorf("当前任务缺少本地缓存文件，无法重传")
	}
	if retryInfo.APIBaseURL == "" {
		return nil, fmt.Errorf("当前任务缺少租户接口地址，无法重传")
	}
	if retryInfo.AccessToken == "" {
		return nil, fmt.Errorf("当前租户缺少登录态，请重新登录后再试")
	}

	fileBytes, err := os.ReadFile(retryInfo.CachedFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取本地缓存文件失败：%w", err)
	}

	uploadInput := model.UploadDesktopMediaInput{
		APIBaseURL:      retryInfo.APIBaseURL,
		AccessToken:     retryInfo.AccessToken,
		FileName:        retryInfo.FileName,
		MimeType:        retryInfo.MimeType,
		FileBase64:      base64.StdEncoding.EncodeToString(fileBytes),
		OriginalName:    retryInfo.OriginalName,
		MediaCategoryID: retryInfo.MediaCategoryID,
		SourceURL:       retryInfo.SourceURL,
		DraftID:         retryInfo.DraftID,
		UploadScene:     retryInfo.UploadScene,
	}
	scope := &model.TenantScopeInfo{
		SiteID:     retryInfo.SiteID,
		TenantID:   retryInfo.TenantID,
		TenantName: "",
	}

	response, err := uploadDesktopMediaRemoteBytes(uploadInput, fileBytes)
	logRequest := model.RemoteDesktopAPIRequestInput{
		APIBaseURL: uploadInput.APIBaseURL,
		Method:     "POST",
		Path:       "/media/upload",
	}
	if logInput := buildAppLogInput(scope, logRequest, response, err); logInput != nil {
		_ = a.store.RecordAppLog(*logInput)
	}
	if mediaTask := buildMediaTaskInput(scope, uploadInput, retryInfo.CachedFilePath, response, err); mediaTask != nil {
		_ = a.store.RecordMediaTask(*mediaTask)
	}
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (a *App) ClearMediaTaskCache(input model.MediaTaskCacheCleanupInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	cacheInfo, err := a.store.GetMediaTaskCacheInfo(input.TaskID)
	if err != nil {
		return err
	}
	if cacheInfo == nil {
		return fmt.Errorf("媒体任务不存在")
	}
	if cacheInfo.CachedFilePath == "" {
		return fmt.Errorf("当前任务没有可清理的本地缓存文件")
	}

	if err := os.Remove(cacheInfo.CachedFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除本地缓存文件失败：%w", err)
	}

	if err := a.store.ClearMediaTaskCacheByPath(cacheInfo.CachedFilePath); err != nil {
		return err
	}

	return nil
}

func (a *App) SaveTenantAuth(input model.SaveTenantAuthInput) (*model.AuthBootstrap, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.SaveTenantAuth(input)
}

func (a *App) ClearSavedTenantToken() error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.ClearSavedTenantToken()
}

func (a *App) DownloadRemoteMedia(input model.DownloadRemoteMediaInput) (*model.DownloadRemoteMediaResult, error) {
	rawURL := strings.TrimSpace(input.URL)
	if rawURL == "" {
		return nil, errors.New("缺少图片地址")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("无效的图片地址：%w", err)
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建下载请求失败：%w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	if referer := strings.TrimSpace(input.Referer); referer != "" {
		req.Header.Set("Referer", referer)
	} else {
		req.Header.Set("Referer", parsedURL.Scheme+"://"+parsedURL.Host+"/")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载图片失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载图片失败，状态码：%d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取图片内容失败：%w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || strings.HasPrefix(contentType, "text/") {
		contentType = "application/octet-stream"
	}

	fileName := path.Base(parsedURL.Path)
	if fileName == "" || fileName == "/" || fileName == "." {
		fileName = "download"
	}

	fileBase64 := base64.StdEncoding.EncodeToString(body)

	return &model.DownloadRemoteMediaResult{
		FileBase64: fileBase64,
		MimeType:   contentType,
		FileName:   fileName,
		FileSize:   int64(len(body)),
	}, nil
}

func (a *App) SaveLocalDraft(input model.SaveLocalDraftInput) (*model.LocalDraftItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.SaveLocalDraft(input)
}

func (a *App) GetLocalDraft(input model.LocalDraftQueryInput) (*model.LocalDraftItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.GetLocalDraft(input)
}

func (a *App) DeleteLocalDraft(input model.LocalDraftQueryInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteLocalDraft(input)
}

func (a *App) ListLocalDrafts(input model.LocalDraftListInput) ([]model.LocalDraftItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListLocalDrafts(input)
}

func (a *App) ListSubmitRecords(input model.SubmitRecordListInput) ([]model.SubmitRecordItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListSubmitRecords(input)
}

func (a *App) DeleteSubmitRecord(jobID int64) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteSubmitRecord(jobID)
}

func (a *App) ListSystemLogs(input model.AppLogListInput) ([]model.AppLogItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListAppLogs(input)
}

func (a *App) DeleteSystemLog(logID int64) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteAppLog(logID)
}

func (a *App) ClearSystemLogs() error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.ClearAppLogs()
}

func (a *App) ListMediaTasks(input model.MediaTaskListInput) ([]model.MediaTaskItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListMediaTasks(input)
}

func (a *App) DeleteMediaTask(taskID int64) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteMediaTask(taskID)
}

func (a *App) recordLocalCollectorLog(input model.CollectWebPageInput, level string, message string, context map[string]any) {
	if a.store == nil {
		return
	}

	rawContext, _ := json.Marshal(context)
	_ = a.store.RecordAppLog(model.AppLogInput{
		SiteID:      input.SiteID,
		TenantID:    input.TenantID,
		Level:       level,
		Module:      "collector.local",
		Message:     message,
		ContextJSON: string(rawContext),
	})
}

func (a *App) CollectWebPage(input model.CollectWebPageInput) (*model.CollectedWebPageResult, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	a.recordLocalCollectorLog(input, "info", "开始本地采集", map[string]any{
		"url": input.URL,
	})

	result, err := collectWebPage(input)
	if err != nil {
		a.recordLocalCollectorLog(input, "error", "本地采集失败", map[string]any{
			"url":   input.URL,
			"error": err.Error(),
		})
		return nil, err
	}

	a.recordLocalCollectorLog(input, "info", "本地采集完成", map[string]any{
		"url":               input.URL,
		"finalUrl":          result.FinalURL,
		"title":             result.Title,
		"imageCount":        len(result.Images),
		"keywordCount":      len(result.Keywords),
		"images":            collectorLogPreviewImages(result.Images, 8),
		"contentHtml":       truncateCollectorLogValue(result.ContentHTML, 1200),
		"contentLength":     len(result.ContentText),
		"contentHtmlLength": len(result.ContentHTML),
	})

	return result, nil
}

func (a *App) RenderWebPagePreview(input model.CollectWebPageInput) (*model.BrowserRenderedPreviewResult, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	a.recordLocalCollectorLog(input, "info", "开始浏览器渲染预览", map[string]any{
		"url": input.URL,
	})

	result, err := renderWebPagePreview(input)
	if err != nil {
		a.recordLocalCollectorLog(input, "error", "浏览器渲染预览失败", map[string]any{
			"url":   input.URL,
			"error": err.Error(),
		})
		return nil, err
	}

	a.recordLocalCollectorLog(input, "info", "浏览器渲染预览完成", map[string]any{
		"url":      input.URL,
		"finalUrl": result.FinalURL,
		"browser":  result.Browser,
	})

	return result, nil
}

func (a *App) ScanSiteLinks(input model.ScanSiteLinksInput) (*model.ScanSiteLinksResult, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	result, err := scanSiteLinks(input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func collectorLogPreviewImages(images []model.CollectedImage, limit int) []string {
	if limit <= 0 || len(images) == 0 {
		return nil
	}
	if len(images) < limit {
		limit = len(images)
	}
	result := make([]string, 0, limit)
	for _, item := range images[:limit] {
		if value := model.FirstNonEmpty(item.URL); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func truncateCollectorLogValue(value string, limit int) string {
	text := strings.TrimSpace(value)
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
