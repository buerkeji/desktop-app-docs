package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"zq-desktop-app/internal/model"
)

type tenantScopeInfo = model.TenantScopeInfo

type SubmitRecordInput = model.SubmitRecordInput
type SubmitRecordListInput = model.SubmitRecordListInput
type SubmitRecordItem = model.SubmitRecordItem
type AppLogInput = model.AppLogInput
type AppLogListInput = model.AppLogListInput
type AppLogItem = model.AppLogItem
type MediaTaskInput = model.MediaTaskInput
type MediaTaskListInput = model.MediaTaskListInput
type MediaTaskItem = model.MediaTaskItem
type MediaTaskRetryInput = model.MediaTaskRetryInput
type MediaTaskCacheCleanupInput = model.MediaTaskCacheCleanupInput
type MediaTaskRetryInfo = model.MediaTaskRetryInfo
type MediaTaskCacheInfo = model.MediaTaskCacheInfo

func (s *DesktopStore) FindTenantScopeByAPIBaseURL(apiBaseURL string) (*tenantScopeInfo, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if baseURL == "" {
		return nil, nil
	}

	row := s.db.QueryRow(
		`SELECT id, site_id, name
		 FROM tenants
		 WHERE RTRIM(api_base_url, '/') = RTRIM(?, '/')
		 LIMIT 1`,
		baseURL,
	)

	var scope tenantScopeInfo
	err := row.Scan(&scope.TenantID, &scope.SiteID, &scope.TenantName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find tenant by api base url: %w", err)
	}

	return &scope, nil
}

func (s *DesktopStore) RecordSubmitRecord(input SubmitRecordInput) error {
	if input.SiteID <= 0 || input.TenantID <= 0 {
		return nil
	}

	contentType := strings.TrimSpace(input.ContentType)
	jobType := strings.TrimSpace(input.JobType)
	if contentType == "" || jobType == "" {
		return nil
	}

	status := model.NormaliseSubmitStatus(input.Status)
	title := strings.TrimSpace(input.Title)
	now := time.Now().Format(time.RFC3339)
	startedAt := model.FirstNonEmptyTimestamp(input.StartedAt, input.FinishedAt, now)
	finishedAt := model.FirstNonEmptyTimestamp(input.FinishedAt, now)
	payloadJSON := model.CompactJSONString(input.PayloadJSON)
	resultJSON := model.CompactJSONString(input.ResultJSON)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin submit record tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.Exec(
		`INSERT INTO submit_jobs (
			site_id, tenant_id, title, content_type, job_type, idempotency_key,
			payload_json, status, started_at, finished_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.SiteID,
		input.TenantID,
		title,
		contentType,
		jobType,
		strings.TrimSpace(input.IdempotencyKey),
		payloadJSON,
		status,
		startedAt,
		finishedAt,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert submit job: %w", err)
	}

	jobID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get submit job id: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO submit_results (
			job_id, draft_id, content_type, remote_id, remote_url, match_type,
			created_count, updated_count, failed_count, result_json, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		jobID,
		model.NullableInt64(input.DraftID),
		contentType,
		model.NullableInt64(input.RemoteID),
		strings.TrimSpace(input.RemoteURL),
		strings.TrimSpace(input.MatchType),
		input.CreatedCount,
		input.UpdatedCount,
		input.FailedCount,
		resultJSON,
		strings.TrimSpace(input.ErrorMessage),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert submit result: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit submit record tx: %w", err)
	}

	return nil
}

func (s *DesktopStore) DeleteMediaTask(taskID int64) error {
	if taskID <= 0 {
		return errors.New("缺少任务 ID")
	}

	_, err := s.db.Exec(
		`DELETE FROM media_tasks WHERE id = ?`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("delete media task: %w", err)
	}

	return nil
}

func (s *DesktopStore) ListSubmitRecords(input SubmitRecordListInput) ([]SubmitRecordItem, error) {
	query := `SELECT
			sr.id,
			sj.id,
			sj.site_id,
			sj.tenant_id,
			COALESCE(t.name, ''),
			sj.title,
			COALESCE(NULLIF(sr.content_type, ''), sj.content_type),
			sj.job_type,
			sj.status,
			sj.idempotency_key,
			COALESCE(sr.remote_id, 0),
			sr.remote_url,
			sr.match_type,
			COALESCE(sr.created_count, 0),
			COALESCE(sr.updated_count, 0),
			COALESCE(sr.failed_count, 0),
			sr.error_message,
			sj.payload_json,
			sr.result_json,
			COALESCE(NULLIF(sj.finished_at, ''), sr.updated_at, sj.updated_at, sj.created_at),
			sj.created_at,
			sj.updated_at
		FROM submit_jobs sj
		LEFT JOIN submit_results sr ON sr.job_id = sj.id
		LEFT JOIN tenants t ON t.id = sj.tenant_id
		WHERE 1 = 1`
	args := make([]any, 0, 8)

	if input.TenantID > 0 {
		query += ` AND sj.tenant_id = ?`
		args = append(args, input.TenantID)
	}
	if keyword := strings.TrimSpace(input.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query += ` AND (sj.title LIKE ? OR sr.error_message LIKE ? OR sj.idempotency_key LIKE ?)`
		args = append(args, like, like, like)
	}
	if contentType := strings.TrimSpace(input.ContentType); contentType != "" {
		query += ` AND COALESCE(NULLIF(sr.content_type, ''), sj.content_type) = ?`
		args = append(args, contentType)
	}
	if status := strings.TrimSpace(input.Status); status != "" {
		query += ` AND sj.status = ?`
		args = append(args, status)
	}
	if dateFrom := model.NormaliseDateBoundary(input.DateFrom, false); dateFrom != "" {
		query += ` AND COALESCE(NULLIF(sj.finished_at, ''), sr.updated_at, sj.updated_at, sj.created_at) >= ?`
		args = append(args, dateFrom)
	}
	if dateTo := model.NormaliseDateBoundary(input.DateTo, true); dateTo != "" {
		query += ` AND COALESCE(NULLIF(sj.finished_at, ''), sr.updated_at, sj.updated_at, sj.created_at) <= ?`
		args = append(args, dateTo)
	}

	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query += ` ORDER BY COALESCE(NULLIF(sj.finished_at, ''), sr.updated_at, sj.updated_at, sj.created_at) DESC, sj.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list submit records: %w", err)
	}
	defer rows.Close()

	items := make([]SubmitRecordItem, 0)
	for rows.Next() {
		var item SubmitRecordItem
		if err := rows.Scan(
			&item.ID,
			&item.JobID,
			&item.SiteID,
			&item.TenantID,
			&item.TenantName,
			&item.Title,
			&item.ContentType,
			&item.JobType,
			&item.Status,
			&item.IdempotencyKey,
			&item.RemoteID,
			&item.RemoteURL,
			&item.MatchType,
			&item.CreatedCount,
			&item.UpdatedCount,
			&item.FailedCount,
			&item.ErrorMessage,
			&item.PayloadJSON,
			&item.ResultJSON,
			&item.SubmittedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan submit record: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *DesktopStore) DeleteSubmitRecord(jobID int64) error {
	if jobID <= 0 {
		return errors.New("缺少任务 ID")
	}

	_, err := s.db.Exec(
		`DELETE FROM submit_results WHERE job_id = ?`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("delete submit result: %w", err)
	}

	_, err = s.db.Exec(
		`DELETE FROM submit_jobs WHERE id = ?`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("delete submit job: %w", err)
	}

	return nil
}

func (s *DesktopStore) RecordAppLog(input AppLogInput) error {
	level := model.NormaliseLogLevel(input.Level)
	module := strings.TrimSpace(input.Module)
	message := strings.TrimSpace(input.Message)
	if module == "" || message == "" {
		return nil
	}

	_, err := s.db.Exec(
		`INSERT INTO app_logs (site_id, tenant_id, request_id, level, module, message, context_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.SiteID,
		input.TenantID,
		strings.TrimSpace(input.RequestID),
		level,
		module,
		message,
		model.CompactJSONString(input.ContextJSON),
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert app log: %w", err)
	}

	return nil
}

func (s *DesktopStore) ListAppLogs(input AppLogListInput) ([]AppLogItem, error) {
	query := `SELECT
			al.id,
			al.site_id,
			al.tenant_id,
			COALESCE(t.name, ''),
			al.request_id,
			al.level,
			al.module,
			al.message,
			al.context_json,
			al.created_at
		FROM app_logs al
		LEFT JOIN tenants t ON t.id = al.tenant_id
		WHERE 1 = 1`
	args := make([]any, 0, 8)

	if input.TenantID > 0 {
		query += ` AND al.tenant_id = ?`
		args = append(args, input.TenantID)
	}
	if keyword := strings.TrimSpace(input.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query += ` AND (al.message LIKE ? OR al.context_json LIKE ?)`
		args = append(args, like, like)
	}
	if level := strings.TrimSpace(input.Level); level != "" {
		query += ` AND al.level = ?`
		args = append(args, level)
	}
	if module := strings.TrimSpace(input.Module); module != "" {
		query += ` AND al.module = ?`
		args = append(args, module)
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		query += ` AND al.request_id LIKE ?`
		args = append(args, "%"+requestID+"%")
	}
	if dateFrom := model.NormaliseDateBoundary(input.DateFrom, false); dateFrom != "" {
		query += ` AND al.created_at >= ?`
		args = append(args, dateFrom)
	}
	if dateTo := model.NormaliseDateBoundary(input.DateTo, true); dateTo != "" {
		query += ` AND al.created_at <= ?`
		args = append(args, dateTo)
	}

	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query += ` ORDER BY al.created_at DESC, al.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list app logs: %w", err)
	}
	defer rows.Close()

	items := make([]AppLogItem, 0)
	for rows.Next() {
		var item AppLogItem
		if err := rows.Scan(
			&item.ID,
			&item.SiteID,
			&item.TenantID,
			&item.TenantName,
			&item.RequestID,
			&item.Level,
			&item.Module,
			&item.Message,
			&item.ContextJSON,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan app log: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *DesktopStore) DeleteAppLog(logID int64) error {
	if logID <= 0 {
		return errors.New("缺少日志 ID")
	}

	_, err := s.db.Exec(
		`DELETE FROM app_logs WHERE id = ?`,
		logID,
	)
	if err != nil {
		return fmt.Errorf("delete app log: %w", err)
	}

	return nil
}

func (s *DesktopStore) ClearAppLogs() error {
	_, err := s.db.Exec(`DELETE FROM app_logs`)
	if err != nil {
		return fmt.Errorf("clear app logs: %w", err)
	}

	return nil
}

func (s *DesktopStore) RecordMediaTask(input MediaTaskInput) error {
	if input.SiteID <= 0 || input.TenantID <= 0 {
		return nil
	}

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO media_tasks (
			site_id, tenant_id, file_name, original_name, mime_type, upload_scene, cached_file_path, source_url, media_category_id, draft_id,
			status, request_id, remote_media_id, remote_url, remote_path, disk, size_bytes, width, height,
			error_message, response_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.SiteID,
		input.TenantID,
		strings.TrimSpace(input.FileName),
		strings.TrimSpace(input.OriginalName),
		strings.TrimSpace(input.MimeType),
		model.NormaliseUploadScene(input.UploadScene),
		strings.TrimSpace(input.CachedFilePath),
		strings.TrimSpace(input.SourceURL),
		input.MediaCategoryID,
		input.DraftID,
		model.NormaliseTaskStatus(input.Status),
		strings.TrimSpace(input.RequestID),
		input.RemoteMediaID,
		strings.TrimSpace(input.RemoteURL),
		strings.TrimSpace(input.RemotePath),
		strings.TrimSpace(input.Disk),
		input.SizeBytes,
		input.Width,
		input.Height,
		strings.TrimSpace(input.ErrorMessage),
		model.CompactJSONString(input.ResponseJSON),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert media task: %w", err)
	}

	return nil
}

func (s *DesktopStore) ListMediaTasks(input MediaTaskListInput) ([]MediaTaskItem, error) {
	query := `SELECT
			mt.id,
			mt.site_id,
			mt.tenant_id,
			COALESCE(t.name, ''),
			mt.file_name,
			mt.original_name,
			mt.mime_type,
			mt.upload_scene,
			mt.cached_file_path,
			mt.source_url,
			mt.media_category_id,
			mt.draft_id,
			mt.status,
			mt.request_id,
			mt.remote_media_id,
			mt.remote_url,
			mt.remote_path,
			mt.disk,
			mt.size_bytes,
			mt.width,
			mt.height,
			mt.error_message,
			mt.response_json,
			mt.created_at,
			mt.updated_at
		FROM media_tasks mt
		LEFT JOIN tenants t ON t.id = mt.tenant_id
		WHERE 1 = 1`
	args := make([]any, 0, 8)

	if input.TenantID > 0 {
		query += ` AND mt.tenant_id = ?`
		args = append(args, input.TenantID)
	}
	if keyword := strings.TrimSpace(input.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query += ` AND (mt.file_name LIKE ? OR mt.original_name LIKE ? OR mt.remote_url LIKE ? OR mt.error_message LIKE ?)`
		args = append(args, like, like, like, like)
	}
	if status := strings.TrimSpace(input.Status); status != "" {
		query += ` AND mt.status = ?`
		args = append(args, status)
	}
	if scene := strings.TrimSpace(input.Scene); scene != "" {
		query += ` AND mt.upload_scene = ?`
		args = append(args, scene)
	}
	if dateFrom := model.NormaliseDateBoundary(input.DateFrom, false); dateFrom != "" {
		query += ` AND mt.created_at >= ?`
		args = append(args, dateFrom)
	}
	if dateTo := model.NormaliseDateBoundary(input.DateTo, true); dateTo != "" {
		query += ` AND mt.created_at <= ?`
		args = append(args, dateTo)
	}

	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query += ` ORDER BY mt.created_at DESC, mt.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list media tasks: %w", err)
	}
	defer rows.Close()

	items := make([]MediaTaskItem, 0)
	for rows.Next() {
		var item MediaTaskItem
		var cachedFilePath string
		if err := rows.Scan(
			&item.ID,
			&item.SiteID,
			&item.TenantID,
			&item.TenantName,
			&item.FileName,
			&item.OriginalName,
			&item.MimeType,
			&item.UploadScene,
			&cachedFilePath,
			&item.SourceURL,
			&item.MediaCategoryID,
			&item.DraftID,
			&item.Status,
			&item.RequestID,
			&item.RemoteMediaID,
			&item.RemoteURL,
			&item.RemotePath,
			&item.Disk,
			&item.SizeBytes,
			&item.Width,
			&item.Height,
			&item.ErrorMessage,
			&item.ResponseJSON,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan media task: %w", err)
		}
		item.CanRetry = strings.TrimSpace(cachedFilePath) != ""
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *DesktopStore) GetMediaTaskRetryInfo(taskID int64) (*model.MediaTaskRetryInfo, error) {
	if taskID <= 0 {
		return nil, errors.New("缺少媒体任务 ID")
	}

	row := s.db.QueryRow(
		`SELECT
			mt.id,
			mt.site_id,
			mt.tenant_id,
			COALESCE(t.api_base_url, ''),
			COALESCE(tt.access_token, ''),
			mt.file_name,
			mt.original_name,
			mt.mime_type,
			mt.upload_scene,
			mt.cached_file_path,
			mt.source_url,
			mt.media_category_id,
			mt.draft_id
		FROM media_tasks mt
		LEFT JOIN tenants t ON t.id = mt.tenant_id
		LEFT JOIN tenant_tokens tt ON tt.tenant_id = mt.tenant_id
		WHERE mt.id = ?
		LIMIT 1`,
		taskID,
	)

	var item model.MediaTaskRetryInfo
	if err := row.Scan(
		&item.TaskID,
		&item.SiteID,
		&item.TenantID,
		&item.APIBaseURL,
		&item.AccessToken,
		&item.FileName,
		&item.OriginalName,
		&item.MimeType,
		&item.UploadScene,
		&item.CachedFilePath,
		&item.SourceURL,
		&item.MediaCategoryID,
		&item.DraftID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("媒体任务不存在")
		}
		return nil, fmt.Errorf("get media task retry info: %w", err)
	}

	return &item, nil
}

func (s *DesktopStore) GetMediaTaskCacheInfo(taskID int64) (*model.MediaTaskCacheInfo, error) {
	if taskID <= 0 {
		return nil, errors.New("缺少媒体任务 ID")
	}

	row := s.db.QueryRow(
		`SELECT id, cached_file_path
		FROM media_tasks
		WHERE id = ?
		LIMIT 1`,
		taskID,
	)

	var item model.MediaTaskCacheInfo
	if err := row.Scan(&item.TaskID, &item.CachedFilePath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("媒体任务不存在")
		}
		return nil, fmt.Errorf("get media task cache info: %w", err)
	}

	return &item, nil
}

func (s *DesktopStore) ClearMediaTaskCacheByPath(cachedFilePath string) error {
	cachedFilePath = strings.TrimSpace(cachedFilePath)
	if cachedFilePath == "" {
		return nil
	}

	_, err := s.db.Exec(
		`UPDATE media_tasks
		SET cached_file_path = '', updated_at = ?
		WHERE cached_file_path = ?`,
		time.Now().Format(time.RFC3339),
		cachedFilePath,
	)
	if err != nil {
		return fmt.Errorf("clear media task cache path: %w", err)
	}

	return nil
}
