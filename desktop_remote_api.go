package main

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zq-desktop-app/internal/model"
)

type desktopAPIEnvelope[T any] struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	Data      T                   `json:"data"`
	Errors    map[string][]string `json:"errors,omitempty"`
	ErrorCode string              `json:"error_code,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}

type remoteTenantDiscoveryResponse struct {
	Site struct {
		Host         string `json:"host"`
		BaseURL      string `json:"base_url"`
		IsSystemHost bool   `json:"is_system_host"`
	} `json:"site"`
	Tenants []struct {
		Slug               string `json:"slug"`
		Name               string `json:"name"`
		HasActiveDomain    bool   `json:"has_active_domain"`
		RecommendedBaseURL string `json:"recommended_base_url"`
		APIBaseURL         string `json:"api_base_url"`
		LoginHint          struct {
			Mode       string `json:"mode"`
			TenantSlug string `json:"tenant_slug"`
		} `json:"login_hint"`
	} `json:"tenants"`
}

type remoteLoginResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	SessionID        int64  `json:"session_id"`
	TokenType        string `json:"token_type"`
	ExpiresAt        string `json:"expires_at"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
	User             struct {
		ID       int64    `json:"id"`
		Username string   `json:"username"`
		Name     string   `json:"name"`
		TenantID int64    `json:"tenant_id"`
		Roles    []string `json:"roles"`
	} `json:"user"`
}

func normaliseRemoteBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func buildRemoteDesktopAPIURL(baseURL string, path string) string {
	base := normaliseRemoteBaseURL(baseURL)
	suffix := path
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}

	return base + "/api/desktop" + suffix
}

func buildRemoteDesktopAPIPathURL(apiBaseURL string, path string) string {
	base := normaliseRemoteBaseURL(apiBaseURL)
	suffix := path
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}

	return base + suffix
}

func newRemoteDesktopAPIRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Client-Type", "desktop")
	req.Header.Set("X-Client-Version", "0.1.0")
	req.Header.Set("X-Request-Id", fmt.Sprintf("desktop-%d", time.Now().UnixNano()))

	return req, nil
}

func desktopAPIProxy(req *http.Request) (*url.URL, error) {
	if req != nil && req.URL != nil {
		host := strings.TrimSpace(strings.ToLower(req.URL.Hostname()))
		if host == "localhost" || strings.HasSuffix(host, ".localhost") {
			return nil, nil
		}

		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil, nil
		}
	}

	return http.ProxyFromEnvironment(req)
}

func remoteDesktopAPIClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = desktopAPIProxy

	return &http.Client{
		Timeout:   20 * time.Second,
		Transport: transport,
	}
}

func appendRemoteRequestID(message string, requestID string) string {
	message = strings.TrimSpace(message)
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return message
	}
	if message == "" {
		return "请求失败，请联系管理员并提供请求 ID：" + requestID
	}

	return fmt.Sprintf("%s [请求ID: %s]", message, requestID)
}

func flattenRemoteValidationErrors(items map[string][]string) string {
	if len(items) == 0 {
		return ""
	}

	messages := make([]string, 0, len(items))
	for _, fieldErrors := range items {
		for _, item := range fieldErrors {
			text := strings.TrimSpace(item)
			if text != "" {
				messages = append(messages, text)
			}
		}
	}

	return strings.Join(messages, "；")
}

func buildRemoteAPIResponseMessage(statusCode int, payload desktopAPIEnvelope[json.RawMessage]) string {
	message := strings.TrimSpace(payload.Message)
	validation := flattenRemoteValidationErrors(payload.Errors)

	if validation != "" {
		if message != "" {
			message = message + "：" + validation
		} else {
			message = validation
		}
	}

	if message == "" {
		switch statusCode {
		case http.StatusUnauthorized:
			message = "登录已失效或当前账号没有访问该接口的权限"
		case http.StatusForbidden:
			message = "当前账号没有权限执行此操作"
		case http.StatusNotFound:
			message = "接口地址不存在，请确认服务端已发布桌面端接口"
		case http.StatusUnprocessableEntity:
			message = "提交的数据未通过校验，请检查填写内容"
		default:
			if statusCode >= http.StatusInternalServerError {
				message = fmt.Sprintf("服务端处理失败（HTTP %d），请稍后重试", statusCode)
			} else {
				message = fmt.Sprintf("接口请求失败（HTTP %d）", statusCode)
			}
		}
	}

	return appendRemoteRequestID(message, payload.RequestID)
}

func buildRemoteInvalidResponseMessage(statusCode int, body string) string {
	snippet := strings.TrimSpace(body)
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}

	if statusCode == http.StatusNotFound {
		return "接口返回了非 JSON 内容，可能是站点地址不正确，或服务端未启用桌面端接口"
	}
	if strings.Contains(strings.ToLower(snippet), "<html") || strings.Contains(strings.ToLower(snippet), "<!doctype") {
		return "接口返回了页面内容而不是 JSON，可能是站点地址不正确，或被登录页/网关拦截"
	}
	if snippet == "" {
		return fmt.Sprintf("接口返回内容无法识别（HTTP %d）", statusCode)
	}

	return fmt.Sprintf("接口返回内容无法识别（HTTP %d）：%s", statusCode, snippet)
}

func describeRemoteRequestError(err error) string {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "无法解析站点域名，请检查站点地址或 DNS 配置"
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return "连接站点超时，请检查网络连通性或服务响应时间"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "连接站点超时，请检查网络连通性或服务响应时间"
	}

	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthorityErr) {
		return "HTTPS 证书不受系统信任，请检查站点证书或改用受信任证书"
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return "HTTPS 证书与站点域名不匹配，请检查访问地址和证书配置"
	}

	var certInvalidErr x509.CertificateInvalidError
	if errors.As(err, &certInvalidErr) {
		return "HTTPS 证书无效，请检查证书是否过期或配置是否正确"
	}

	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "connection refused"), strings.Contains(lower, "actively refused"):
		return "无法连接到站点，请确认服务已启动并允许当前电脑访问"
	case strings.Contains(lower, "no such host"):
		return "无法解析站点域名，请检查站点地址是否正确"
	case strings.Contains(lower, "certificate"):
		return "HTTPS 证书校验失败，请检查证书配置"
	case strings.Contains(lower, "tls"):
		return "TLS 握手失败，请检查 HTTPS 协议与证书配置"
	case strings.Contains(lower, "deadline exceeded"):
		return "请求站点超时，请稍后重试"
	default:
		return "请求站点失败：" + strings.TrimSpace(err.Error())
	}
}

func executeRemoteDesktopAPIRequest(
	method string,
	url string,
	headers map[string]string,
	body io.Reader,
	accessToken string,
) (*model.RemoteDesktopAPIResponse, error) {
	req, err := newRemoteDesktopAPIRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	}

	resp, err := remoteDesktopAPIClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s", describeRemoteRequestError(err))
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	var payload desktopAPIEnvelope[json.RawMessage]
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, fmt.Errorf("%s", buildRemoteInvalidResponseMessage(resp.StatusCode, string(rawBody)))
	}

	var data any
	if len(payload.Data) > 0 && string(payload.Data) != "null" {
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			data = string(payload.Data)
		}
	}

	return &model.RemoteDesktopAPIResponse{
		Success:    payload.Success && resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices,
		Message:    buildRemoteAPIResponseMessage(resp.StatusCode, payload),
		Data:       data,
		Errors:     payload.Errors,
		ErrorCode:  payload.ErrorCode,
		RequestID:  payload.RequestID,
		StatusCode: resp.StatusCode,
	}, nil
}

func proxyDesktopAPIRequest(input model.RemoteDesktopAPIRequestInput) (*model.RemoteDesktopAPIResponse, error) {
	apiBaseURL := normaliseRemoteBaseURL(input.APIBaseURL)
	if apiBaseURL == "" {
		return nil, fmt.Errorf("缺少桌面端 API 地址")
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if input.Body != "" {
		body = strings.NewReader(input.Body)
	}

	response, err := executeRemoteDesktopAPIRequest(
		method,
		buildRemoteDesktopAPIPathURL(apiBaseURL, input.Path),
		input.Headers,
		body,
		input.AccessToken,
	)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func uploadDesktopMediaRemote(input model.UploadDesktopMediaInput) (*model.RemoteDesktopAPIResponse, error) {
	apiBaseURL := normaliseRemoteBaseURL(input.APIBaseURL)
	if apiBaseURL == "" {
		return nil, fmt.Errorf("缺少桌面端 API 地址")
	}
	if strings.TrimSpace(input.AccessToken) == "" {
		return nil, fmt.Errorf("缺少访问令牌，请重新登录")
	}
	if strings.TrimSpace(input.FileName) == "" {
		return nil, fmt.Errorf("缺少上传文件名")
	}
	if strings.TrimSpace(input.FileBase64) == "" {
		return nil, fmt.Errorf("缺少上传文件内容")
	}

	fileBytes, err := decodeDesktopUploadFileBase64(input.FileBase64)
	if err != nil {
		return nil, err
	}

	return uploadDesktopMediaRemoteBytes(input, fileBytes)
}

func decodeDesktopUploadFileBase64(fileBase64 string) ([]byte, error) {
	fileBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(fileBase64))
	if err != nil {
		return nil, fmt.Errorf("解析上传文件失败：%w", err)
	}
	return fileBytes, nil
}

func uploadDesktopMediaRemoteBytes(input model.UploadDesktopMediaInput, fileBytes []byte) (*model.RemoteDesktopAPIResponse, error) {
	apiBaseURL := normaliseRemoteBaseURL(input.APIBaseURL)
	if apiBaseURL == "" {
		return nil, fmt.Errorf("缺少桌面端 API 地址")
	}
	if strings.TrimSpace(input.AccessToken) == "" {
		return nil, fmt.Errorf("缺少访问令牌，请重新登录")
	}
	if strings.TrimSpace(input.FileName) == "" {
		return nil, fmt.Errorf("缺少上传文件名")
	}
	if len(fileBytes) == 0 {
		return nil, fmt.Errorf("缺少上传文件内容")
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", input.FileName)
	if err != nil {
		return nil, fmt.Errorf("创建上传请求失败：%w", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("写入上传文件失败：%w", err)
	}

	if strings.TrimSpace(input.OriginalName) != "" {
		_ = writer.WriteField("original_name", strings.TrimSpace(input.OriginalName))
	}
	if input.MediaCategoryID > 0 {
		_ = writer.WriteField("media_category_id", fmt.Sprintf("%d", input.MediaCategoryID))
	}
	if strings.TrimSpace(input.SourceURL) != "" {
		_ = writer.WriteField("source_url", strings.TrimSpace(input.SourceURL))
	}
	if input.DraftID > 0 {
		_ = writer.WriteField("draft_id", fmt.Sprintf("%d", input.DraftID))
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("生成上传表单失败：%w", err)
	}

	response, err := executeRemoteDesktopAPIRequest(
		http.MethodPost,
		buildRemoteDesktopAPIPathURL(apiBaseURL, "/media/upload"),
		map[string]string{
			"Content-Type": writer.FormDataContentType(),
		},
		&requestBody,
		input.AccessToken,
	)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func discoverSiteTenantsRemote(input model.DiscoverSiteTenantsInput) (*model.TenantDiscoveryPayload, error) {
	baseURL := normaliseRemoteBaseURL(input.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("缺少站点地址")
	}

	response, err := executeRemoteDesktopAPIRequest(
		http.MethodGet,
		buildRemoteDesktopAPIURL(baseURL, "/tenants/discovery"),
		nil,
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}

	if !response.Success {
		message := model.FirstNonEmpty(response.Message, "同步租户失败")
		return nil, fmt.Errorf("%s", message)
	}

	var payload remoteTenantDiscoveryResponse
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("解析租户数据失败：%w", err)
	}
	if err := json.Unmarshal(dataBytes, &payload); err != nil {
		return nil, fmt.Errorf("解析租户数据失败：%w", err)
	}

	result := &model.TenantDiscoveryPayload{
		Site: model.SiteDiscoveryInfo{
			Host:         payload.Site.Host,
			BaseURL:      payload.Site.BaseURL,
			IsSystemHost: payload.Site.IsSystemHost,
		},
		Tenants: make([]model.TenantDiscoveryItem, 0, len(payload.Tenants)),
	}

	for _, item := range payload.Tenants {
		result.Tenants = append(result.Tenants, model.TenantDiscoveryItem{
			Slug:               item.Slug,
			Name:               item.Name,
			HasActiveDomain:    item.HasActiveDomain,
			RecommendedBaseURL: item.RecommendedBaseURL,
			APIBaseURL:         item.APIBaseURL,
			LoginHint: model.TenantDiscoveryLoginHint{
				Mode:       item.LoginHint.Mode,
				TenantSlug: item.LoginHint.TenantSlug,
			},
		})
	}

	return result, nil
}

func loginTenantRemote(input model.LoginTenantInput) (*remoteLoginResponse, error) {
	apiBaseURL := normaliseRemoteBaseURL(input.APIBaseURL)
	if apiBaseURL == "" {
		return nil, fmt.Errorf("缺少桌面端 API 地址")
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, fmt.Errorf("缺少用户名")
	}
	password := strings.TrimSpace(input.Password)
	if password == "" {
		return nil, fmt.Errorf("缺少密码")
	}

	requestBody, err := json.Marshal(map[string]any{
		"username":       username,
		"password":       password,
		"device_name":    strings.TrimSpace(input.DeviceName),
		"client_version": strings.TrimSpace(input.ClientVersion),
		"tenant_slug":    strings.TrimSpace(input.TenantSlug),
	})
	if err != nil {
		return nil, fmt.Errorf("构造登录请求失败：%w", err)
	}

	response, err := executeRemoteDesktopAPIRequest(
		http.MethodPost,
		buildRemoteDesktopAPIPathURL(apiBaseURL, "/auth/login"),
		map[string]string{
			"Content-Type": "application/json",
		},
		bytes.NewReader(requestBody),
		"",
	)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		message := model.FirstNonEmpty(response.Message, "登录失败")
		return nil, fmt.Errorf("%s", message)
	}

	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("解析登录结果失败：%w", err)
	}

	var payload remoteLoginResponse
	if err := json.Unmarshal(dataBytes, &payload); err != nil {
		return nil, fmt.Errorf("解析登录结果失败：%w", err)
	}

	return &payload, nil
}

func refreshTenantTokenRemote(input model.RefreshTenantTokenInput) (*remoteLoginResponse, error) {
	apiBaseURL := normaliseRemoteBaseURL(input.APIBaseURL)
	if apiBaseURL == "" {
		return nil, fmt.Errorf("缺少桌面端 API 地址")
	}
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("缺少刷新令牌，请重新登录")
	}

	requestBody, err := json.Marshal(map[string]any{
		"refresh_token":  refreshToken,
		"device_name":    strings.TrimSpace(input.DeviceName),
		"client_version": strings.TrimSpace(input.ClientVersion),
	})
	if err != nil {
		return nil, fmt.Errorf("构造刷新请求失败：%w", err)
	}

	response, err := executeRemoteDesktopAPIRequest(
		http.MethodPost,
		buildRemoteDesktopAPIPathURL(apiBaseURL, "/auth/refresh"),
		map[string]string{
			"Content-Type": "application/json",
		},
		bytes.NewReader(requestBody),
		"",
	)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		message := model.FirstNonEmpty(response.Message, "刷新登录状态失败")
		return nil, fmt.Errorf("%s", message)
	}

	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("解析刷新结果失败：%w", err)
	}

	var payload remoteLoginResponse
	if err := json.Unmarshal(dataBytes, &payload); err != nil {
		return nil, fmt.Errorf("解析刷新结果失败：%w", err)
	}

	return &payload, nil
}

func logoutTenantRemote(input model.LogoutTenantInput) error {
	apiBaseURL := normaliseRemoteBaseURL(input.APIBaseURL)
	if apiBaseURL == "" {
		return fmt.Errorf("缺少桌面端 API 地址")
	}
	if strings.TrimSpace(input.AccessToken) == "" {
		return fmt.Errorf("缺少访问令牌，请重新登录")
	}

	response, err := executeRemoteDesktopAPIRequest(
		http.MethodPost,
		buildRemoteDesktopAPIPathURL(apiBaseURL, "/auth/logout"),
		nil,
		nil,
		input.AccessToken,
	)
	if err != nil {
		return err
	}
	if !response.Success {
		message := model.FirstNonEmpty(response.Message, "退出登录失败")
		return fmt.Errorf("%s", message)
	}

	return nil
}
