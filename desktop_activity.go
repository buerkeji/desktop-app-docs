package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zq-desktop-app/internal/model"
)

type submitOperationMeta struct {
	Title          string
	ContentType    string
	JobType        string
	IdempotencyKey string
	PayloadJSON    string
}

func buildMediaTaskInput(scope *model.TenantScopeInfo, input model.UploadDesktopMediaInput, cachedFilePath string, response *model.RemoteDesktopAPIResponse, requestErr error) *model.MediaTaskInput {
	if scope == nil || scope.SiteID <= 0 || scope.TenantID <= 0 {
		return nil
	}

	task := &model.MediaTaskInput{
		SiteID:          scope.SiteID,
		TenantID:        scope.TenantID,
		FileName:        input.FileName,
		OriginalName:    model.FirstNonEmpty(input.OriginalName, input.FileName),
		MimeType:        input.MimeType,
		UploadScene:     input.UploadScene,
		CachedFilePath:  strings.TrimSpace(cachedFilePath),
		SourceURL:       input.SourceURL,
		MediaCategoryID: input.MediaCategoryID,
		DraftID:         input.DraftID,
		Status:          "success",
		RequestID:       model.ResponseRequestID(response),
	}

	if requestErr != nil {
		task.Status = "failed"
		task.ErrorMessage = requestErr.Error()
		return task
	}
	if response == nil {
		task.Status = "failed"
		task.ErrorMessage = "empty response"
		return task
	}
	if !response.Success {
		task.Status = "failed"
		task.ErrorMessage = strings.TrimSpace(response.Message)
		task.ResponseJSON = marshalJSON(response.Data)
		return task
	}

	task.ResponseJSON = marshalJSON(response.Data)
	data := model.ToStringAnyMap(response.Data)
	task.RemoteMediaID = model.Int64FromAny(data["media_item_id"])
	task.RemoteURL = strings.TrimSpace(fmt.Sprint(data["url"]))
	task.RemotePath = strings.TrimSpace(fmt.Sprint(data["path"]))
	task.Disk = strings.TrimSpace(fmt.Sprint(data["disk"]))
	task.SizeBytes = model.Int64FromAny(data["size"])
	task.Width = model.Int64FromAny(data["width"])
	task.Height = model.Int64FromAny(data["height"])

	return task
}

func buildAppLogInput(scope *model.TenantScopeInfo, input model.RemoteDesktopAPIRequestInput, response *model.RemoteDesktopAPIResponse, requestErr error) *model.AppLogInput {
	if !shouldPersistAppLog(input, response, requestErr) {
		return nil
	}

	module := classifyRemoteModule(input.Path)
	message := fmt.Sprintf("%s %s", strings.ToUpper(strings.TrimSpace(input.Method)), model.FirstNonEmpty(normaliseRequestPath(input.Path), "/"))
	level := "info"

	if requestErr != nil {
		level = "error"
		message = requestErr.Error()
	} else if response != nil && !response.Success {
		level = "error"
		if strings.TrimSpace(response.Message) != "" {
			message = response.Message
		}
	} else if response != nil && strings.TrimSpace(response.Message) != "" {
		message = response.Message
	}

	context := map[string]any{
		"method": strings.ToUpper(strings.TrimSpace(input.Method)),
		"path":   normaliseRequestPath(input.Path),
	}
	if response != nil {
		context["success"] = response.Success
		context["message"] = response.Message
		context["requestId"] = response.RequestID
	}
	if requestErr != nil {
		context["error"] = requestErr.Error()
	}
	if payload := model.CompactJSONString(input.Body); payload != "" {
		context["requestBody"] = json.RawMessage(payload)
	}
	if response != nil && response.Data != nil {
		context["responseData"] = response.Data
	}

	rawContext, _ := json.Marshal(context)
	logInput := &model.AppLogInput{
		Level:       level,
		Module:      module,
		Message:     message,
		RequestID:   model.FirstNonEmpty(model.ResponseRequestID(response), model.RequestIDFromHeaders(input.Headers)),
		ContextJSON: string(rawContext),
	}
	if scope != nil {
		logInput.SiteID = scope.SiteID
		logInput.TenantID = scope.TenantID
	}

	return logInput
}

func buildSubmitRecordInput(scope *model.TenantScopeInfo, input model.RemoteDesktopAPIRequestInput, response *model.RemoteDesktopAPIResponse, requestErr error) *model.SubmitRecordInput {
	meta := detectSubmitOperation(input)
	if meta == nil || scope == nil || scope.SiteID <= 0 || scope.TenantID <= 0 {
		return nil
	}

	now := time.Now().Format(time.RFC3339)
	record := &model.SubmitRecordInput{
		SiteID:         scope.SiteID,
		TenantID:       scope.TenantID,
		Title:          meta.Title,
		ContentType:    meta.ContentType,
		JobType:        meta.JobType,
		IdempotencyKey: meta.IdempotencyKey,
		PayloadJSON:    meta.PayloadJSON,
		Status:         "success",
		StartedAt:      now,
		FinishedAt:     now,
	}

	if requestErr != nil {
		record.Status = "failed"
		record.ErrorMessage = requestErr.Error()
		return record
	}

	if response == nil {
		record.Status = "failed"
		record.ErrorMessage = "empty response"
		return record
	}

	record.ResultJSON = marshalJSON(response.Data)
	if !response.Success {
		record.Status = "failed"
		record.ErrorMessage = strings.TrimSpace(response.Message)
		return record
	}

	switch meta.JobType {
	case "create":
		record.CreatedCount = 1
	case "update":
		record.UpdatedCount = 1
	case "batch_delete":
		deleted, failed := extractBatchDeleteCounts(response.Data)
		record.UpdatedCount = deleted
		record.FailedCount = failed
		if deleted > 0 && failed > 0 {
			record.Status = "partial"
		}
	}

	if dataMap := model.ToStringAnyMap(response.Data); len(dataMap) > 0 {
		record.Title = model.FirstNonEmpty(fmt.Sprint(dataMap["title"]), record.Title)
		record.RemoteID = model.Int64FromAny(dataMap["id"])
		record.RemoteURL = strings.TrimSpace(fmt.Sprint(dataMap["url"]))
		record.MatchType = strings.TrimSpace(fmt.Sprint(dataMap["match_type"]))
	}

	return record
}

func detectSubmitOperation(input model.RemoteDesktopAPIRequestInput) *submitOperationMeta {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	path := normaliseRequestPath(input.Path)
	body := strings.TrimSpace(input.Body)
	idempotencyKey := model.RequestIDFromHeaders(input.Headers)

	switch {
	case method == "POST" && path == "/tools":
		return &submitOperationMeta{
			Title:          titleFromRequestBody(body),
			ContentType:    "tool",
			JobType:        "create",
			IdempotencyKey: idempotencyKey,
			PayloadJSON:    body,
		}
	case method == "PUT" && strings.HasPrefix(path, "/tools/"):
		return &submitOperationMeta{
			Title:          titleFromRequestBody(body),
			ContentType:    "tool",
			JobType:        "update",
			IdempotencyKey: idempotencyKey,
			PayloadJSON:    body,
		}
	case method == "POST" && path == "/articles":
		return &submitOperationMeta{
			Title:          titleFromRequestBody(body),
			ContentType:    "article",
			JobType:        "create",
			IdempotencyKey: idempotencyKey,
			PayloadJSON:    body,
		}
	case method == "PUT" && strings.HasPrefix(path, "/articles/"):
		return &submitOperationMeta{
			Title:          titleFromRequestBody(body),
			ContentType:    "article",
			JobType:        "update",
			IdempotencyKey: idempotencyKey,
			PayloadJSON:    body,
		}
	case method == "POST" && path == "/tools/batch-delete":
		return &submitOperationMeta{
			Title:          "批量删除工具",
			ContentType:    "tool",
			JobType:        "batch_delete",
			IdempotencyKey: idempotencyKey,
			PayloadJSON:    body,
		}
	case method == "POST" && path == "/articles/batch-delete":
		return &submitOperationMeta{
			Title:          "批量删除文章",
			ContentType:    "article",
			JobType:        "batch_delete",
			IdempotencyKey: idempotencyKey,
			PayloadJSON:    body,
		}
	default:
		return nil
	}
}

func shouldPersistAppLog(input model.RemoteDesktopAPIRequestInput, response *model.RemoteDesktopAPIResponse, requestErr error) bool {
	if requestErr != nil {
		return true
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	module := classifyRemoteModule(input.Path)
	if response != nil && !response.Success {
		return true
	}
	if method != "GET" {
		return true
	}
	return module == "auth" || module == "session" || module == "bootstrap"
}

func classifyRemoteModule(path string) string {
	path = normaliseRequestPath(path)
	switch {
	case strings.HasPrefix(path, "/tools"):
		return "content.tool"
	case strings.HasPrefix(path, "/articles"):
		return "content.article"
	case strings.HasPrefix(path, "/media"):
		return "media.upload"
	case strings.HasPrefix(path, "/auth/sessions"):
		return "session"
	case strings.HasPrefix(path, "/auth"):
		return "auth"
	case strings.Contains(path, "category") || strings.Contains(path, "tag"):
		return "dictionary"
	case strings.HasPrefix(path, "/bootstrap"):
		return "bootstrap"
	default:
		return "desktop.api"
	}
}

func normaliseRequestPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func titleFromRequestBody(body string) string {
	payload := model.ToStringAnyMap(body)
	title := strings.TrimSpace(fmt.Sprint(payload["title"]))
	if title != "" && title != "<nil>" {
		return title
	}
	return ""
}

func extractBatchDeleteCounts(data any) (int, int) {
	root := model.ToStringAnyMap(data)
	summary := model.ToStringAnyMap(root["summary"])
	deleted := int(model.Int64FromAny(summary["deleted"]))
	failed := int(model.Int64FromAny(summary["failed"]))
	return deleted, failed
}

func marshalJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
