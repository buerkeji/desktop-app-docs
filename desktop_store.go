package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zq-desktop-app/internal/model"

	_ "modernc.org/sqlite"
)

var ErrStoreNotReady = errors.New("desktop store not ready")

const encryptedTokenPrefix = "enc:v1:"

type DesktopStore struct {
	db      *sql.DB
	dataDir string
}

func NewDesktopStore() (*DesktopStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config dir: %w", err)
	}

	dataDir := filepath.Join(configDir, "zq-desktop-app")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "desktop.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	store := &DesktopStore{
		db:      db,
		dataDir: dataDir,
	}

	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *DesktopStore) GetDesktopBootstrap() (*model.DesktopBootstrap, error) {
	sites, err := s.listSites()
	if err != nil {
		return nil, err
	}

	tenants, err := s.listTenants()
	if err != nil {
		return nil, err
	}

	token, user, err := s.getSavedAuth()
	if err != nil {
		return nil, err
	}

	allAuths, err := s.listSavedAuths()
	if err != nil {
		return nil, err
	}

	return &model.DesktopBootstrap{
		Sites:       sites,
		Tenants:     tenants,
		Token:       token,
		User:        user,
		TenantAuths: allAuths,
	}, nil
}

func (s *DesktopStore) CreateSite(input model.CreateSiteInput) (*model.SiteItem, error) {
	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimSpace(input.BaseURL)
	if name == "" || baseURL == "" {
		return nil, errors.New("站点名称和站点地址不能为空")
	}

	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(
		`INSERT INTO sites (name, base_url, description, is_default, created_at, updated_at) VALUES (?, ?, ?, 0, ?, ?)`,
		name,
		baseURL,
		strings.TrimSpace(input.Description),
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create site: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get site id: %w", err)
	}

	return &model.SiteItem{
		ID:          id,
		Name:        name,
		BaseURL:     baseURL,
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
	}, nil
}

func (s *DesktopStore) UpdateSite(input model.UpdateSiteInput) (*model.SiteItem, error) {
	if input.ID <= 0 {
		return nil, errors.New("缺少站点 ID")
	}

	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimSpace(input.BaseURL)
	if name == "" || baseURL == "" {
		return nil, errors.New("站点名称和站点地址不能为空")
	}

	result, err := s.db.Exec(
		`UPDATE sites
		 SET name = ?, base_url = ?, description = ?, updated_at = ?
		 WHERE id = ?`,
		name,
		baseURL,
		strings.TrimSpace(input.Description),
		time.Now().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update site: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get updated site rows: %w", err)
	}
	if affected == 0 {
		return nil, errors.New("站点不存在或已被删除")
	}

	return s.getSiteByID(input.ID)
}

func (s *DesktopStore) DeleteSite(input model.DeleteSiteInput) error {
	if input.ID <= 0 {
		return errors.New("缺少站点 ID")
	}

	if _, err := s.db.Exec(
		`DELETE FROM tenant_tokens
		 WHERE tenant_id IN (SELECT id FROM tenants WHERE site_id = ?)`,
		input.ID,
	); err != nil {
		return fmt.Errorf("delete site tokens: %w", err)
	}

	result, err := s.db.Exec(`DELETE FROM sites WHERE id = ?`, input.ID)
	if err != nil {
		return fmt.Errorf("delete site: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted site rows: %w", err)
	}
	if affected == 0 {
		return errors.New("站点不存在或已被删除")
	}

	return nil
}

func (s *DesktopStore) CreateTenant(input model.CreateTenantInput) (*model.TenantItem, error) {
	if input.SiteID <= 0 {
		return nil, errors.New("缺少所属站点")
	}

	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimSpace(input.BaseURL)
	apiBaseURL := strings.TrimSpace(input.APIBaseURL)
	tenantName := strings.TrimSpace(input.TenantName)
	tenantSlug := strings.TrimSpace(input.TenantSlug)
	lastUsername := strings.TrimSpace(input.LastUsername)
	if name == "" || baseURL == "" || apiBaseURL == "" {
		return nil, errors.New("租户名称、域名和 API 地址不能为空")
	}

	now := time.Now().Format(time.RFC3339)
	capabilitiesJSON := model.MustJSON(defaultTenantCapabilities())
	limitsJSON := model.MustJSON(defaultTenantLimits())

	_, err := s.db.Exec(
		`INSERT INTO tenants (site_id, name, base_url, api_base_url, tenant_name, tenant_slug, last_username, status, capabilities_json, limits_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_id, base_url, tenant_slug) DO UPDATE SET
		   name = excluded.name,
		   api_base_url = excluded.api_base_url,
		   tenant_name = CASE
		     WHEN excluded.tenant_name <> '' THEN excluded.tenant_name
		     ELSE tenants.tenant_name
		   END,
		   last_username = CASE
		     WHEN excluded.last_username <> '' THEN excluded.last_username
		     ELSE tenants.last_username
		   END,
		   status = excluded.status,
		   updated_at = excluded.updated_at`,
		input.SiteID,
		name,
		baseURL,
		apiBaseURL,
		tenantName,
		tenantSlug,
		lastUsername,
		"enabled",
		capabilitiesJSON,
		limitsJSON,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return s.getTenantBySiteBaseAndSlug(input.SiteID, baseURL, tenantSlug)
}

func (s *DesktopStore) UpdateTenant(input model.UpdateTenantInput) (*model.TenantItem, error) {
	if input.ID <= 0 {
		return nil, errors.New("缺少租户 ID")
	}
	if input.SiteID <= 0 {
		return nil, errors.New("缺少所属站点")
	}

	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimSpace(input.BaseURL)
	apiBaseURL := strings.TrimSpace(input.APIBaseURL)
	if name == "" || baseURL == "" || apiBaseURL == "" {
		return nil, errors.New("租户名称、域名和 API 地址不能为空")
	}

	result, err := s.db.Exec(
		`UPDATE tenants
		 SET site_id = ?, name = ?, base_url = ?, api_base_url = ?, tenant_name = ?, tenant_slug = ?, last_username = ?, updated_at = ?
		 WHERE id = ?`,
		input.SiteID,
		name,
		baseURL,
		apiBaseURL,
		strings.TrimSpace(input.TenantName),
		strings.TrimSpace(input.TenantSlug),
		strings.TrimSpace(input.LastUsername),
		time.Now().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get updated tenant rows: %w", err)
	}
	if affected == 0 {
		return nil, errors.New("租户不存在或已被删除")
	}

	return s.getTenantByID(input.ID)
}

func (s *DesktopStore) DeleteTenant(input model.DeleteTenantInput) error {
	if input.ID <= 0 {
		return errors.New("缺少租户 ID")
	}

	if _, err := s.db.Exec(`DELETE FROM tenant_tokens WHERE tenant_id = ?`, input.ID); err != nil {
		return fmt.Errorf("delete tenant tokens: %w", err)
	}

	result, err := s.db.Exec(`DELETE FROM tenants WHERE id = ?`, input.ID)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted tenant rows: %w", err)
	}
	if affected == 0 {
		return errors.New("租户不存在或已被删除")
	}

	return nil
}

func (s *DesktopStore) getTenantBySiteBaseAndSlug(siteID int64, baseURL string, tenantSlug string) (*model.TenantItem, error) {
	row := s.db.QueryRow(
		`SELECT id, site_id, name, base_url, api_base_url, tenant_name, tenant_slug, last_username, status, capabilities_json, limits_json, created_at
		 FROM tenants
		 WHERE site_id = ? AND base_url = ? AND tenant_slug = ?
		 LIMIT 1`,
		siteID,
		baseURL,
		tenantSlug,
	)

	var item model.TenantItem
	var capabilitiesRaw string
	var limitsRaw string
	err := row.Scan(
		&item.ID,
		&item.SiteID,
		&item.Name,
		&item.BaseURL,
		&item.APIBaseURL,
		&item.TenantName,
		&item.TenantSlug,
		&item.LastUsername,
		&item.Status,
		&capabilitiesRaw,
		&limitsRaw,
		&item.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("tenant not found after create")
	}
	if err != nil {
		return nil, fmt.Errorf("query tenant after create: %w", err)
	}

	item.Capabilities = parseTenantCapabilities(capabilitiesRaw)
	item.Limits = parseTenantLimits(limitsRaw)

	return &item, nil
}

func (s *DesktopStore) getTenantByID(tenantID int64) (*model.TenantItem, error) {
	row := s.db.QueryRow(
		`SELECT id, site_id, name, base_url, api_base_url, tenant_name, tenant_slug, last_username, status, capabilities_json, limits_json, created_at
		 FROM tenants
		 WHERE id = ?
		 LIMIT 1`,
		tenantID,
	)

	var item model.TenantItem
	var capabilitiesRaw string
	var limitsRaw string
	err := row.Scan(
		&item.ID,
		&item.SiteID,
		&item.Name,
		&item.BaseURL,
		&item.APIBaseURL,
		&item.TenantName,
		&item.TenantSlug,
		&item.LastUsername,
		&item.Status,
		&capabilitiesRaw,
		&limitsRaw,
		&item.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("租户不存在或已被删除")
	}
	if err != nil {
		return nil, fmt.Errorf("query tenant: %w", err)
	}

	item.Capabilities = parseTenantCapabilities(capabilitiesRaw)
	item.Limits = parseTenantLimits(limitsRaw)
	return &item, nil
}

func (s *DesktopStore) SaveTenantAuth(input model.SaveTenantAuthInput) (*model.AuthBootstrap, error) {
	if input.TenantID <= 0 {
		return nil, errors.New("tenant id is required")
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, errors.New("username is required")
	}

	if strings.TrimSpace(input.AccessToken) == "" {
		return nil, errors.New("access token is required")
	}

	now := time.Now()
	token := model.TenantTokenState{
		TenantID:         input.TenantID,
		AccessToken:      strings.TrimSpace(input.AccessToken),
		RefreshToken:     strings.TrimSpace(input.RefreshToken),
		TokenType:        strings.TrimSpace(input.TokenType),
		ExpiresAt:        strings.TrimSpace(input.ExpiresAt),
		RefreshExpiresAt: strings.TrimSpace(input.RefreshExpiresAt),
		SessionID:        input.SessionID,
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	if token.ExpiresAt == "" {
		token.ExpiresAt = now.Add(8 * time.Hour).Format(time.RFC3339)
	}

	user := model.CurrentUser{
		ID:       input.UserID,
		Name:     strings.TrimSpace(input.Name),
		Username: username,
		Roles:    input.Roles,
	}
	if user.ID <= 0 {
		user.ID = 1
	}
	if user.Name == "" {
		user.Name = username
	}
	if len(user.Roles) == 0 {
		user.Roles = []string{"desktop"}
	}

	rolesJSON := model.MustJSON(user.Roles)
	updatedAt := now.Format(time.RFC3339)
	accessTokenStored, err := s.encryptStoredToken(token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}

	refreshTokenStored, err := s.encryptStoredToken(token.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt refresh token: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO tenant_tokens (tenant_id, access_token, refresh_token, token_type, expires_at, refresh_expires_at, session_id, user_id, username, user_name, roles_json, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(tenant_id) DO UPDATE SET
		   access_token = excluded.access_token,
		   refresh_token = excluded.refresh_token,
		   token_type = excluded.token_type,
		   expires_at = excluded.expires_at,
		   refresh_expires_at = excluded.refresh_expires_at,
		   session_id = excluded.session_id,
		   user_id = excluded.user_id,
		   username = excluded.username,
		   user_name = excluded.user_name,
		   roles_json = excluded.roles_json,
		   updated_at = excluded.updated_at`,
		token.TenantID,
		accessTokenStored,
		refreshTokenStored,
		token.TokenType,
		token.ExpiresAt,
		token.RefreshExpiresAt,
		token.SessionID,
		user.ID,
		user.Username,
		user.Name,
		rolesJSON,
		updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	_, err = s.db.Exec(`UPDATE tenants SET last_username = ?, updated_at = ? WHERE id = ?`, user.Username, updatedAt, token.TenantID)
	if err != nil {
		return nil, fmt.Errorf("update tenant username: %w", err)
	}

	return &model.AuthBootstrap{
		Token: token,
		User:  user,
	}, nil
}

func (s *DesktopStore) ClearSavedTenantToken() error {
	_, err := s.db.Exec(`DELETE FROM tenant_tokens`)
	if err != nil {
		return fmt.Errorf("clear token: %w", err)
	}

	return nil
}

func (s *DesktopStore) ClearTenantToken(tenantID int64) error {
	if tenantID <= 0 {
		return nil
	}

	_, err := s.db.Exec(`DELETE FROM tenant_tokens WHERE tenant_id = ?`, tenantID)
	if err != nil {
		return fmt.Errorf("clear tenant token: %w", err)
	}

	return nil
}

func (s *DesktopStore) SaveLocalDraft(input model.SaveLocalDraftInput) (*model.LocalDraftItem, error) {
	tenantID, contentType, targetID, err := validateLocalDraftScope(input.TenantID, input.ContentType, input.TargetID)
	if err != nil {
		return nil, err
	}

	payloadJSON := strings.TrimSpace(input.PayloadJSON)
	if payloadJSON == "" {
		return nil, errors.New("draft payload is required")
	}

	now := time.Now().Format(time.RFC3339)
	title := strings.TrimSpace(input.Title)

	_, err = s.db.Exec(
		`INSERT INTO local_drafts (tenant_id, content_type, target_id, title, payload_json, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(tenant_id, content_type, target_id) DO UPDATE SET
		   title = excluded.title,
		   payload_json = excluded.payload_json,
		   updated_at = excluded.updated_at`,
		tenantID,
		contentType,
		targetID,
		title,
		payloadJSON,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("save local draft: %w", err)
	}

	return s.GetLocalDraft(model.LocalDraftQueryInput{
		TenantID:    tenantID,
		ContentType: contentType,
		TargetID:    targetID,
	})
}

func (s *DesktopStore) GetLocalDraft(input model.LocalDraftQueryInput) (*model.LocalDraftItem, error) {
	tenantID, contentType, targetID, err := validateLocalDraftScope(input.TenantID, input.ContentType, input.TargetID)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRow(
		`SELECT id, tenant_id, content_type, target_id, title, payload_json, updated_at
		 FROM local_drafts
		 WHERE tenant_id = ? AND content_type = ? AND target_id = ?
		 LIMIT 1`,
		tenantID,
		contentType,
		targetID,
	)

	var item model.LocalDraftItem
	err = row.Scan(&item.ID, &item.TenantID, &item.ContentType, &item.TargetID, &item.Title, &item.PayloadJSON, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get local draft: %w", err)
	}

	return &item, nil
}

func (s *DesktopStore) DeleteLocalDraft(input model.LocalDraftQueryInput) error {
	tenantID, contentType, targetID, err := validateLocalDraftScope(input.TenantID, input.ContentType, input.TargetID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`DELETE FROM local_drafts WHERE tenant_id = ? AND content_type = ? AND target_id = ?`,
		tenantID,
		contentType,
		targetID,
	)
	if err != nil {
		return fmt.Errorf("delete local draft: %w", err)
	}

	return nil
}

func (s *DesktopStore) ListLocalDrafts(input model.LocalDraftListInput) ([]model.LocalDraftItem, error) {
	if input.TenantID <= 0 {
		return nil, errors.New("tenant id is required")
	}

	contentType := strings.TrimSpace(input.ContentType)
	query := `SELECT id, tenant_id, content_type, target_id, title, payload_json, updated_at
		FROM local_drafts
		WHERE tenant_id = ?`
	args := []any{input.TenantID}
	if contentType != "" {
		query += ` AND content_type = ?`
		args = append(args, contentType)
	}
	query += ` ORDER BY updated_at DESC, id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list local drafts: %w", err)
	}
	defer rows.Close()

	items := make([]model.LocalDraftItem, 0)
	for rows.Next() {
		var item model.LocalDraftItem
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ContentType, &item.TargetID, &item.Title, &item.PayloadJSON, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan local draft: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *DesktopStore) listSites() ([]model.SiteItem, error) {
	rows, err := s.db.Query(`SELECT id, name, base_url, description, is_default, created_at FROM sites ORDER BY is_default DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("query sites: %w", err)
	}
	defer rows.Close()

	var items []model.SiteItem
	for rows.Next() {
		var item model.SiteItem
		var isDefault int
		if err := rows.Scan(&item.ID, &item.Name, &item.BaseURL, &item.Description, &isDefault, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan site: %w", err)
		}

		item.IsDefault = isDefault == 1
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *DesktopStore) getSiteByID(siteID int64) (*model.SiteItem, error) {
	row := s.db.QueryRow(
		`SELECT id, name, base_url, description, is_default, created_at
		 FROM sites
		 WHERE id = ?
		 LIMIT 1`,
		siteID,
	)

	var item model.SiteItem
	var isDefault int
	if err := row.Scan(&item.ID, &item.Name, &item.BaseURL, &item.Description, &isDefault, &item.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("站点不存在或已被删除")
		}
		return nil, fmt.Errorf("query site: %w", err)
	}

	item.IsDefault = isDefault == 1
	return &item, nil
}

func (s *DesktopStore) listTenants() ([]model.TenantItem, error) {
	rows, err := s.db.Query(`SELECT id, site_id, name, base_url, api_base_url, tenant_name, tenant_slug, last_username, status, capabilities_json, limits_json, created_at FROM tenants ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	var items []model.TenantItem
	for rows.Next() {
		var item model.TenantItem
		var capabilitiesRaw string
		var limitsRaw string
		if err := rows.Scan(
			&item.ID,
			&item.SiteID,
			&item.Name,
			&item.BaseURL,
			&item.APIBaseURL,
			&item.TenantName,
			&item.TenantSlug,
			&item.LastUsername,
			&item.Status,
			&capabilitiesRaw,
			&limitsRaw,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}

		item.Capabilities = parseTenantCapabilities(capabilitiesRaw)
		item.Limits = parseTenantLimits(limitsRaw)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *DesktopStore) getSavedAuth() (*model.TenantTokenState, *model.CurrentUser, error) {
	row := s.db.QueryRow(`SELECT tenant_id, access_token, refresh_token, token_type, expires_at, refresh_expires_at, session_id, user_id, username, user_name, roles_json FROM tenant_tokens ORDER BY updated_at DESC LIMIT 1`)

	var token model.TenantTokenState
	var user model.CurrentUser
	var rolesRaw string
	var accessTokenStored string
	var refreshTokenStored string

	err := row.Scan(
		&token.TenantID,
		&accessTokenStored,
		&refreshTokenStored,
		&token.TokenType,
		&token.ExpiresAt,
		&token.RefreshExpiresAt,
		&token.SessionID,
		&user.ID,
		&user.Username,
		&user.Name,
		&rolesRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query token: %w", err)
	}

	token.AccessToken, err = s.decryptStoredToken(accessTokenStored)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt access token: %w", err)
	}

	token.RefreshToken, err = s.decryptStoredToken(refreshTokenStored)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt refresh token: %w", err)
	}

	user.ID = 1
	if err := json.Unmarshal([]byte(rolesRaw), &user.Roles); err != nil {
		user.Roles = []string{"admin"}
	}

	if !s.isEncryptedStoredToken(accessTokenStored) || (!s.isEncryptedStoredToken(refreshTokenStored) && token.RefreshToken != "") {
		_ = s.reencryptStoredTenantToken(token.TenantID, token.AccessToken, token.RefreshToken)
	}

	return &token, &user, nil
}

func (s *DesktopStore) listSavedAuths() ([]model.TenantAuthEntry, error) {
	rows, err := s.db.Query(`SELECT tenant_id, access_token, refresh_token, token_type, expires_at, refresh_expires_at, session_id, user_id, username, user_name, roles_json FROM tenant_tokens ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query all tokens: %w", err)
	}
	defer rows.Close()

	var entries []model.TenantAuthEntry
	for rows.Next() {
		var entry model.TenantAuthEntry
		var rolesRaw string
		var accessTokenStored string
		var refreshTokenStored string

		if err := rows.Scan(
			&entry.TenantID,
			&accessTokenStored,
			&refreshTokenStored,
			&entry.Token.TokenType,
			&entry.Token.ExpiresAt,
			&entry.Token.RefreshExpiresAt,
			&entry.Token.SessionID,
			&entry.User.ID,
			&entry.User.Username,
			&entry.User.Name,
			&rolesRaw,
		); err != nil {
			return nil, fmt.Errorf("scan token row: %w", err)
		}

		entry.Token.AccessToken, err = s.decryptStoredToken(accessTokenStored)
		if err != nil {
			continue
		}
		entry.Token.RefreshToken, err = s.decryptStoredToken(refreshTokenStored)
		if err != nil {
			continue
		}
		entry.Token.TenantID = entry.TenantID

		entry.User.ID = 1
		if err := json.Unmarshal([]byte(rolesRaw), &entry.User.Roles); err != nil {
			entry.User.Roles = []string{"admin"}
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (s *DesktopStore) encryptStoredToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	protected, err := s.protectSecret([]byte(value))
	if err != nil {
		return "", err
	}

	return encryptedTokenPrefix + base64.RawStdEncoding.EncodeToString(protected), nil
}

func (s *DesktopStore) decryptStoredToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	if !s.isEncryptedStoredToken(value) {
		return value, nil
	}

	protected, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, encryptedTokenPrefix))
	if err != nil {
		return "", fmt.Errorf("decode protected token: %w", err)
	}

	plain, err := s.unprotectSecret(protected)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func (s *DesktopStore) isEncryptedStoredToken(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), encryptedTokenPrefix)
}

func (s *DesktopStore) reencryptStoredTenantToken(tenantID int64, accessToken string, refreshToken string) error {
	accessTokenStored, err := s.encryptStoredToken(accessToken)
	if err != nil {
		return err
	}

	refreshTokenStored, err := s.encryptStoredToken(refreshToken)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`UPDATE tenant_tokens SET access_token = ?, refresh_token = ? WHERE tenant_id = ?`,
		accessTokenStored,
		refreshTokenStored,
		tenantID,
	)

	return err
}

func (s *DesktopStore) migrate() error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS sites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tenants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			api_base_url TEXT NOT NULL,
			tenant_name TEXT NOT NULL DEFAULT '',
			tenant_slug TEXT NOT NULL DEFAULT '',
			last_username TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'enabled',
			capabilities_json TEXT NOT NULL,
			limits_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE
		);`,
		`DROP INDEX IF EXISTS idx_tenants_site_base;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_site_base_slug ON tenants(site_id, base_url, tenant_slug);`,
		`CREATE TABLE IF NOT EXISTS tenant_tokens (
			tenant_id INTEGER PRIMARY KEY,
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			token_type TEXT NOT NULL DEFAULT 'Bearer',
			expires_at TEXT NOT NULL,
			refresh_expires_at TEXT NOT NULL DEFAULT '',
			session_id INTEGER NOT NULL DEFAULT 0,
			user_id INTEGER NOT NULL DEFAULT 0,
			username TEXT NOT NULL,
			user_name TEXT NOT NULL,
			roles_json TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS local_drafts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			content_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			payload_json TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_local_drafts_scope ON local_drafts(tenant_id, content_type, target_id);`,
		`CREATE TABLE IF NOT EXISTS submit_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			tenant_id INTEGER NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			content_type TEXT NOT NULL,
			job_type TEXT NOT NULL,
			idempotency_key TEXT NOT NULL DEFAULT '',
			payload_json TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE,
			FOREIGN KEY(tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_submit_jobs_tenant_id ON submit_jobs(tenant_id);`,
		`CREATE INDEX IF NOT EXISTS idx_submit_jobs_idempotency_key ON submit_jobs(idempotency_key);`,
		`CREATE INDEX IF NOT EXISTS idx_submit_jobs_status ON submit_jobs(status);`,
		`CREATE TABLE IF NOT EXISTS submit_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			draft_id INTEGER,
			content_type TEXT NOT NULL DEFAULT '',
			remote_id INTEGER,
			remote_url TEXT NOT NULL DEFAULT '',
			match_type TEXT NOT NULL DEFAULT '',
			created_count INTEGER NOT NULL DEFAULT 0,
			updated_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			result_json TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(job_id) REFERENCES submit_jobs(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_submit_results_job_id ON submit_results(job_id);`,
		`CREATE TABLE IF NOT EXISTS app_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL DEFAULT 0,
			tenant_id INTEGER NOT NULL DEFAULT 0,
			request_id TEXT NOT NULL DEFAULT '',
			level TEXT NOT NULL,
			module TEXT NOT NULL,
			message TEXT NOT NULL,
			context_json TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_app_logs_level ON app_logs(level);`,
		`CREATE INDEX IF NOT EXISTS idx_app_logs_module ON app_logs(module);`,
		`CREATE INDEX IF NOT EXISTS idx_app_logs_request_id ON app_logs(request_id);`,
		`CREATE INDEX IF NOT EXISTS idx_app_logs_tenant_id ON app_logs(tenant_id);`,
		`CREATE TABLE IF NOT EXISTS media_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			tenant_id INTEGER NOT NULL,
			file_name TEXT NOT NULL DEFAULT '',
			original_name TEXT NOT NULL DEFAULT '',
			mime_type TEXT NOT NULL DEFAULT '',
			upload_scene TEXT NOT NULL DEFAULT '',
			cached_file_path TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			media_category_id INTEGER NOT NULL DEFAULT 0,
			draft_id INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			request_id TEXT NOT NULL DEFAULT '',
			remote_media_id INTEGER NOT NULL DEFAULT 0,
			remote_url TEXT NOT NULL DEFAULT '',
			remote_path TEXT NOT NULL DEFAULT '',
			disk TEXT NOT NULL DEFAULT '',
			size_bytes INTEGER NOT NULL DEFAULT 0,
			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			response_json TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE,
			FOREIGN KEY(tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_media_tasks_tenant_id ON media_tasks(tenant_id);`,
		`CREATE INDEX IF NOT EXISTS idx_media_tasks_status ON media_tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_media_tasks_request_id ON media_tasks(request_id);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("migrate schema: %w", err)
		}
	}

	alterStatements := []string{
		`ALTER TABLE tenant_tokens ADD COLUMN token_type TEXT NOT NULL DEFAULT 'Bearer';`,
		`ALTER TABLE tenant_tokens ADD COLUMN refresh_expires_at TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE tenant_tokens ADD COLUMN session_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE tenant_tokens ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE submit_jobs ADD COLUMN title TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_jobs ADD COLUMN idempotency_key TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_jobs ADD COLUMN payload_json TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_results ADD COLUMN content_type TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_results ADD COLUMN remote_url TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_results ADD COLUMN match_type TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_results ADD COLUMN created_count INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE submit_results ADD COLUMN updated_count INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE submit_results ADD COLUMN failed_count INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE submit_results ADD COLUMN result_json TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE submit_results ADD COLUMN error_message TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE app_logs ADD COLUMN site_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE app_logs ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE app_logs ADD COLUMN request_id TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE app_logs ADD COLUMN context_json TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN upload_scene TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN cached_file_path TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN source_url TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN media_category_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE media_tasks ADD COLUMN draft_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE media_tasks ADD COLUMN request_id TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN remote_media_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE media_tasks ADD COLUMN remote_url TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN remote_path TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN disk TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN size_bytes INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE media_tasks ADD COLUMN width INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE media_tasks ADD COLUMN height INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE media_tasks ADD COLUMN error_message TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE media_tasks ADD COLUMN response_json TEXT NOT NULL DEFAULT '';`,
	}

	for _, statement := range alterStatements {
		if _, err := s.db.Exec(statement); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return fmt.Errorf("migrate tenant_tokens columns: %w", err)
		}
	}

	return nil
}

func (s *DesktopStore) SaveMediaUploadCache(fileName string, fileBytes []byte) (string, error) {
	if len(fileBytes) == 0 {
		return "", nil
	}

	cacheDir := filepath.Join(s.dataDir, "media-cache", time.Now().Format("200601"))
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create media cache dir: %w", err)
	}

	ext := filepath.Ext(strings.TrimSpace(fileName))
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(fileName)), ext)
	base = sanitiseCacheFileName(base)
	if base == "" {
		base = "upload"
	}
	if ext == "" {
		ext = ".bin"
	}

	filePath := filepath.Join(cacheDir, fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), base, ext))
	if err := os.WriteFile(filePath, fileBytes, 0o644); err != nil {
		return "", fmt.Errorf("write media cache file: %w", err)
	}

	return filePath, nil
}

func sanitiseCacheFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	builder := strings.Builder{}
	for _, char := range value {
		switch {
		case (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9'):
			builder.WriteRune(char)
		case char == '-' || char == '_':
			builder.WriteRune(char)
		default:
			builder.WriteRune('-')
		}
	}

	return strings.Trim(builder.String(), "-")
}

func defaultTenantCapabilities() model.TenantCapabilities {
	return model.TenantCapabilities{
		Article:    true,
		Tool:       true,
		Dictionary: true,
		Media:      true,
	}
}

func defaultTenantLimits() model.TenantLimits {
	return model.TenantLimits{
		MaxUploadMB:   5,
		MaxBatchCount: 50,
	}
}

func parseTenantCapabilities(raw string) model.TenantCapabilities {
	result := defaultTenantCapabilities()
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func parseTenantLimits(raw string) model.TenantLimits {
	result := defaultTenantLimits()
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func validateLocalDraftScope(tenantID int64, contentType string, targetID string) (int64, string, string, error) {
	if tenantID <= 0 {
		return 0, "", "", errors.New("tenant id is required")
	}

	contentType = strings.TrimSpace(contentType)
	targetID = strings.TrimSpace(targetID)
	if contentType == "" {
		return 0, "", "", errors.New("content type is required")
	}
	if targetID == "" {
		return 0, "", "", errors.New("target id is required")
	}

	return tenantID, contentType, targetID, nil
}
