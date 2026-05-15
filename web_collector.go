package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	stdhtml "html"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"zq-desktop-app/internal/model"

	"github.com/gorilla/websocket"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type CollectWebPageInput = model.CollectWebPageInput
type CollectedImage = model.CollectedImage
type CollectedWebPageResult = model.CollectedWebPageResult
type BrowserRenderedPreviewResult = model.BrowserRenderedPreviewResult
type ScanSiteLinksInput = model.ScanSiteLinksInput
type ScanFilterRule = model.ScanFilterRule
type SiteLinkItem = model.SiteLinkItem
type ScanSiteLinksResult = model.ScanSiteLinksResult

type collectorMeta struct {
	title       string
	description string
	siteName    string
	keywords    []string
	tags        []string
	icon        string
	ogImage     string
	canonical   string
	publishedAt string
}

type collectorCandidate struct {
	node      *xhtml.Node
	score     int
	textLen   int
	depth     int
	innerHTML string
	text      string
}

type leadImageCandidate struct {
	image *CollectedImage
	score int
	order int
}

type collectorImageResolver struct {
	pageURL *neturl.URL
	client  *http.Client
	cache   map[string]string
}

func collectWebPage(input CollectWebPageInput) (*CollectedWebPageResult, error) {
	pageURL, err := normaliseCollectorURL(input.URL)
	if err != nil {
		return nil, err
	}

	if parsedURL, parseErr := neturl.Parse(pageURL); parseErr == nil && parsedURL != nil && isWeChatCollectorHost(parsedURL.Hostname()) {
		return collectWeChatPage(input)
	}

	if input.UseBrowserRender {
		renderedPreview, renderErr := renderWebPagePreview(input)
		if renderErr != nil {
			return nil, fmt.Errorf("浏览器渲染失败：%s", renderErr.Error())
		}
		renderedURL, parseErr := neturl.Parse(renderedPreview.FinalURL)
		if parseErr != nil || renderedURL == nil {
			return nil, fmt.Errorf("浏览器渲染结果地址无效：%s", renderedPreview.FinalURL)
		}
		return collectWebPageFromHTML(pageURL, renderedURL, []byte(renderedPreview.HTML), renderedPreview.HTML)
	}

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建采集请求失败：%w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) ZQDesktop/0.1.0")
	req.Header.Set("Cache-Control", "no-cache")

	client := remoteDesktopAPIClient()
	client.Timeout = 25 * time.Second

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s", describeRemoteRequestError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			renderedPreview, renderErr := renderWebPagePreview(input)
			if renderErr == nil {
				renderedURL, parseErr := neturl.Parse(renderedPreview.FinalURL)
				if parseErr == nil && renderedURL != nil {
					return collectWebPageFromHTML(pageURL, renderedURL, []byte(renderedPreview.HTML), renderedPreview.HTML)
				}
			}
		}
		return nil, fmt.Errorf("采集页面失败（HTTP %d）", resp.StatusCode)
	}

	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if contentType != "" && !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml+xml") {
		return nil, fmt.Errorf("目标地址返回的不是 HTML 页面，当前内容类型为 %s", contentType)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil {
		return nil, fmt.Errorf("读取页面内容失败：%w", err)
	}
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("页面没有返回可采集内容")
	}

	return collectWebPageFromHTML(pageURL, resp.Request.URL, bodyBytes, "")
}

func collectWebPageFromHTML(pageURL string, baseURL *neturl.URL, bodyBytes []byte, browserPreviewHTML string) (*CollectedWebPageResult, error) {
	root, err := xhtml.Parse(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("解析页面 HTML 失败：%w", err)
	}

	imageResolver := newCollectorImageResolver(baseURL)
	meta := extractCollectorMeta(root, baseURL)
	body := findHTMLNode(root, atom.Body)
	if body == nil {
		return nil, fmt.Errorf("页面缺少可解析的正文区域")
	}

	contentNode := findBestCollectorContentNode(body, baseURL)
	contentHTML := strings.TrimSpace(renderCollectorChildren(contentNode, baseURL, imageResolver, 0))
	contentText := collapseCollectorWhitespace(extractCollectorText(contentNode))
	if contentHTML == "" {
		contentHTML = strings.TrimSpace(renderCollectorChildren(body, baseURL, imageResolver, 0))
	}
	if contentText == "" {
		contentText = collapseCollectorWhitespace(extractCollectorText(body))
	}
	if contentHTML == "" && contentText == "" {
		return nil, fmt.Errorf("页面正文为空，暂时无法生成草稿")
	}
	meta.tags = collectorUniqueKeywords(append(meta.tags, extractCollectorVisibleTags(contentNode, body)...))

	title := firstNonEmpty(meta.title, extractHeadingText(contentNode), extractHeadingText(body))

	description := firstNonEmpty(meta.description, firstParagraphText(contentNode), firstParagraphText(body))
	excerpt := description

	images := collectCollectorImages(contentNode, baseURL, imageResolver)
	iconURL := firstNonEmpty(
		imageResolver.resolve(meta.icon),
		imageResolver.resolve(meta.ogImage),
	)
	thumbnailURL := imageResolver.resolve(meta.ogImage)
	if thumbnailURL == "" {
		if thumbnail := collectCollectorLeadImage(contentNode, body, baseURL, imageResolver, images); thumbnail != nil {
			thumbnailURL = thumbnail.URL
		}
	}
	if thumbnailURL == "" && len(images) > 0 {
		thumbnailURL = strings.TrimSpace(images[0].URL)
	}
	officialURL := collectCollectorOfficialURL(body, baseURL)
	excludedKeywordSources := []string{meta.siteName, baseURL.Hostname()}
	seoKeywords := collectorUniqueKeywords(meta.keywords, excludedKeywordSources...)
	suggestedTags := collectorUniqueKeywords(append(append([]string{}, meta.tags...), seoKeywords...), excludedKeywordSources...)
	if images == nil {
		images = make([]CollectedImage, 0)
	}

	return &CollectedWebPageResult{
		RequestedURL:       pageURL,
		FinalURL:           baseURL.String(),
		CanonicalURL:       meta.canonical,
		OfficialURL:        officialURL,
		Host:               baseURL.Hostname(),
		SourceHTML:         string(bodyBytes),
		BrowserPreviewHTML: browserPreviewHTML,
		SiteName:           meta.siteName,
		Title:              title,
		Description:        description,
		Excerpt:            excerpt,
		IconURL:            iconURL,
		ThumbnailURL:       thumbnailURL,
		ContentHTML:        contentHTML,
		ContentText:        contentText,
		Keywords:           seoKeywords,
		SeoTitle:           firstNonEmpty(meta.title, title),
		SeoDescription:     firstNonEmpty(meta.description, description, excerpt),
		SeoKeywords:        seoKeywords,
		SuggestedTags:      suggestedTags,
		Images:             images,
		PublishedAt:        meta.publishedAt,
		FetchedAt:          time.Now().Format(time.RFC3339),
	}, nil
}

func renderWebPagePreview(input CollectWebPageInput) (*BrowserRenderedPreviewResult, error) {
	pageURL, err := normaliseCollectorURL(input.URL)
	if err != nil {
		return nil, err
	}

	browserPath, browserName, err := findCollectorBrowserExecutable()
	if err != nil {
		return nil, err
	}

	result, cdpErr := renderWebPagePreviewCDP(browserPath, browserName, pageURL)
	if cdpErr != nil {
		return nil, fmt.Errorf("浏览器渲染预览失败：%s", cdpErr.Error())
	}
	return result, nil
}

func readDevToolsPort(ctx context.Context, userDataDir string) string {
	activePortPath := filepath.Join(userDataDir, "DevToolsActivePort")

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ""
		default:
		}
		data, err := os.ReadFile(activePortPath)
		if err == nil {
			lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
			if len(lines) > 0 {
				port := strings.TrimSpace(lines[0])
				if port != "" {
					return port
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ""
}

func killBrowserProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	cmd.Process.Kill()
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func renderWebPagePreviewCDP(browserPath string, browserName string, pageURL string) (*BrowserRenderedPreviewResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	userDataDir, err := os.MkdirTemp("", "zq-desktop-browser-cdp-*")
	if err != nil {
		return nil, fmt.Errorf("创建浏览器临时目录失败：%w", err)
	}
	defer os.RemoveAll(userDataDir)

	args := []string{
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--disable-extensions",
		"--disable-sync",
		"--disable-background-networking",
		"--disable-popup-blocking",
		"--disable-background-timer-throttling",
		"--disable-breakpad",
		"--disable-crashpad-for-testing",
		"--disable-component-update",
		"--disable-features=TranslateUI",
		"--mute-audio",
		"--hide-scrollbars",
		"--remote-debugging-port=0",
		"--user-data-dir=" + userDataDir,
	}

	cmd := exec.CommandContext(ctx, browserPath, args...)
	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("启动浏览器失败（路径：%s）：%w", browserPath, startErr)
	}

	var browserKilled bool
	defer func() {
		if !browserKilled {
			killBrowserProcess(cmd)
		}
	}()

	port := readDevToolsPort(ctx, userDataDir)
	if port == "" {
		return nil, fmt.Errorf("等待浏览器 DevTools 端口超时")
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%s", port)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	devtoolsReady := false
	for attempt := 0; attempt < 20; attempt++ {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("等待 DevTools 就绪超时")
		}
		resp, httpErr := httpClient.Get(baseURL + "/json/version")
		if httpErr == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			devtoolsReady = true
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !devtoolsReady {
		return nil, fmt.Errorf("DevTools 未能在超时内就绪")
	}

	encodedURL := neturl.QueryEscape(pageURL)
	putReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+"/json/new?"+encodedURL, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("创建标签页请求失败：%w", reqErr)
	}
	tabResp, httpErr := httpClient.Do(putReq)
	if httpErr != nil {
		return nil, fmt.Errorf("创建标签页失败：%w", httpErr)
	}
	tabBody, _ := io.ReadAll(io.LimitReader(tabResp.Body, 64<<10))
	tabResp.Body.Close()
	if tabResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("创建标签页 HTTP %d", tabResp.StatusCode)
	}

	var tabInfo struct {
		ID                   string `json:"id"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if jsonErr := json.Unmarshal(tabBody, &tabInfo); jsonErr != nil || tabInfo.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("解析标签页信息失败")
	}

	wsDialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	wsConn, _, wsErr := wsDialer.DialContext(ctx, tabInfo.WebSocketDebuggerURL, nil)
	if wsErr != nil {
		httpClient.Get(baseURL + "/json/close/" + tabInfo.ID)
		return nil, fmt.Errorf("连接 DevTools WebSocket 失败：%w", wsErr)
	}

	cleanup := func() {
		wsConn.Close()
		httpClient.Get(baseURL + "/json/close/" + tabInfo.ID)
	}

	cdpSendWithBody := func(id int, method string, params map[string]any) error {
		msg := map[string]any{"id": id, "method": method}
		if params != nil {
			msg["params"] = params
		}
		if wErr := wsConn.SetWriteDeadline(time.Now().Add(10 * time.Second)); wErr != nil {
			return wErr
		}
		return wsConn.WriteJSON(msg)
	}

	readResult := func(targetID int) (map[string]any, error) {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if err := wsConn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
				return nil, err
			}
			var msg map[string]any
			if err := wsConn.ReadJSON(&msg); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					continue
				}
				return nil, err
			}
			if id, ok := msg["id"].(float64); ok && int(id) == targetID {
				return msg, nil
			}
		}
		return nil, fmt.Errorf("等待 CDP 响应 %d 超时", targetID)
	}

	waitForEvent := func(event string) error {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if err := wsConn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
				return err
			}
			var msg map[string]any
			if err := wsConn.ReadJSON(&msg); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					continue
				}
				return err
			}
			if method, ok := msg["method"].(string); ok && method == event {
				return nil
			}
		}
		return fmt.Errorf("等待事件 %s 超时", event)
	}

	if sErr := cdpSendWithBody(1, "Page.enable", nil); sErr != nil {
		cleanup()
		return nil, fmt.Errorf("启用 Page 域失败：%w", sErr)
	}
	if _, rErr := readResult(1); rErr != nil {
		cleanup()
		return nil, fmt.Errorf("Page.enable 无响应：%w", rErr)
	}

	if nErr := cdpSendWithBody(2, "Page.navigate", map[string]any{"url": pageURL}); nErr != nil {
		cleanup()
		return nil, fmt.Errorf("导航失败：%w", nErr)
	}
	if _, nRespErr := readResult(2); nRespErr != nil {
		cleanup()
		return nil, fmt.Errorf("Page.navigate 无响应：%w", nRespErr)
	}

	loadTimedOut := false
	if eErr := waitForEvent("Page.loadEventFired"); eErr != nil {
		loadTimedOut = true
	}

	if eErr := cdpSendWithBody(3, "Runtime.evaluate", map[string]any{
		"expression":    "document.documentElement.outerHTML",
		"returnByValue": true,
	}); eErr != nil {
		cleanup()
		return nil, fmt.Errorf("执行 HTML 提取失败：%w", eErr)
	}

	evalResp, rErr := readResult(3)
	if rErr != nil {
		if loadTimedOut {
			cleanup()
			return nil, fmt.Errorf("页面加载超时，无法提取 HTML")
		}
		cleanup()
		return nil, fmt.Errorf("提取 HTML 无响应：%w", rErr)
	}

	cleanup()
	killBrowserProcess(cmd)
	browserKilled = true

	resultVal := ""
	if result, ok := evalResp["result"].(map[string]any); ok {
		if rv, ok := result["result"].(map[string]any); ok {
			if v, ok := rv["value"].(string); ok {
				resultVal = strings.TrimSpace(v)
			}
		}
	}

	if resultVal == "" {
		return nil, fmt.Errorf("浏览器渲染提取的 HTML 为空")
	}
	if !strings.Contains(strings.ToLower(resultVal), "<html") {
		resultVal = "<html>" + resultVal + "</html>"
	}

	parsed, parseErr := neturl.Parse(pageURL)
	if parseErr != nil || parsed == nil {
		return nil, fmt.Errorf("页面地址无效")
	}

	return &BrowserRenderedPreviewResult{
		RequestedURL: pageURL,
		FinalURL:     pageURL,
		Host:         parsed.Hostname(),
		Browser:      browserName,
		HTML:         resultVal,
		RenderedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

func findCollectorBrowserExecutable() (string, string, error) {
	type browserCandidate struct {
		name string
		path string
	}

	localAppData := os.Getenv("LOCALAPPDATA")

	candidates := []browserCandidate{
		{name: "Microsoft Edge", path: "msedge.exe"},
		{name: "Google Chrome", path: "chrome.exe"},
		{name: "Google Chrome", path: "google-chrome"},
		{name: "Google Chrome", path: "google-chrome-stable"},
		{name: "Microsoft Edge", path: "microsoft-edge"},

		{name: "Microsoft Edge", path: filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe")},
		{name: "Microsoft Edge", path: filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe")},
		{name: "Google Chrome", path: filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe")},
		{name: "Google Chrome", path: filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe")},
	}

	if localAppData != "" {
		candidates = append(candidates,
			browserCandidate{name: "Microsoft Edge", path: filepath.Join(localAppData, "Microsoft", "Edge", "Application", "msedge.exe")},
			browserCandidate{name: "Google Chrome", path: filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe")},
		)
	}

	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		path := strings.TrimSpace(candidate.path)
		if path == "" {
			continue
		}
		if _, exists := seen[strings.ToLower(path)]; exists {
			continue
		}
		seen[strings.ToLower(path)] = struct{}{}

		if strings.EqualFold(filepath.Base(path), path) {
			resolvedPath, err := exec.LookPath(path)
			if err == nil {
				absPath, _ := filepath.Abs(resolvedPath)
				return absPath, candidate.name, nil
			}
			continue
		}

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			absPath, _ := filepath.Abs(path)
			return absPath, candidate.name, nil
		}
	}

	return "", "", fmt.Errorf("当前系统未找到可用的 Edge 或 Chrome 浏览器，暂时无法启用浏览器渲染点选")
}

func normaliseCollectorURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("请输入要采集的网址")
	}
	if strings.HasPrefix(value, "//") {
		value = "https:" + value
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := neturl.Parse(value)
	if err != nil || parsed == nil {
		return "", fmt.Errorf("请输入有效的网址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("仅支持采集 http 或 https 页面")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("请输入完整的网址")
	}

	return parsed.String(), nil
}

func extractCollectorMeta(root *xhtml.Node, baseURL *neturl.URL) collectorMeta {
	meta := collectorMeta{}

	walkCollectorNode(root, func(node *xhtml.Node, _ int) bool {
		if node.Type != xhtml.ElementNode {
			return true
		}

		switch node.DataAtom {
		case atom.Title:
			if meta.title == "" {
				meta.title = collapseCollectorWhitespace(extractCollectorText(node))
			}
		case atom.Meta:
			name := strings.ToLower(strings.TrimSpace(attrValue(node, "name")))
			property := strings.ToLower(strings.TrimSpace(attrValue(node, "property")))
			itemprop := strings.ToLower(strings.TrimSpace(attrValue(node, "itemprop")))
			content := collapseCollectorWhitespace(attrValue(node, "content"))
			if content == "" {
				return true
			}

			switch {
			case name == "description" || property == "og:description":
				if meta.description == "" {
					meta.description = content
				}
			case property == "og:title" || name == "twitter:title":
				if meta.title == "" {
					meta.title = content
				}
			case property == "og:site_name":
				if meta.siteName == "" {
					meta.siteName = content
				}
			case property == "og:image" || name == "twitter:image" || name == "twitter:image:src" || itemprop == "image" || itemprop == "thumbnailurl":
				if meta.ogImage == "" {
					meta.ogImage = resolveCollectorURL(baseURL, content)
				}
			case name == "keywords" || name == "news_keywords" || name == "parsely-tags" || itemprop == "keywords":
				if len(meta.keywords) == 0 {
					meta.keywords = splitCollectorKeywords(content)
				} else {
					meta.keywords = appendCollectorKeywords(meta.keywords, splitCollectorKeywords(content)...)
				}
			case property == "article:tag" || property == "book:tag" || property == "og:article:tag":
				meta.tags = appendCollectorKeywords(meta.tags, splitCollectorKeywords(content)...)
			case property == "article:section" || property == "book:section":
				meta.tags = appendCollectorKeywords(meta.tags, splitCollectorKeywords(content)...)
			case property == "article:published_time" || name == "pubdate" || name == "publishdate":
				if meta.publishedAt == "" {
					meta.publishedAt = content
				}
			}
		case atom.Link:
			rel := strings.ToLower(strings.TrimSpace(attrValue(node, "rel")))
			href := strings.TrimSpace(attrValue(node, "href"))
			if meta.canonical == "" && collectorLinkHasRel(rel, "canonical") && href != "" {
				meta.canonical = resolveCollectorURL(baseURL, href)
			}
		case atom.Time:
			if meta.publishedAt == "" {
				if datetime := collapseCollectorWhitespace(attrValue(node, "datetime")); datetime != "" {
					meta.publishedAt = datetime
				}
			}
		case atom.Script:
			scriptType := strings.ToLower(strings.TrimSpace(attrValue(node, "type")))
			if scriptType == "application/ld+json" {
				extractCollectorJSONLDMeta(extractCollectorText(node), &meta, baseURL)
			}
		}

		return true
	})

	meta.keywords = collectorUniqueKeywords(meta.keywords)
	meta.tags = collectorUniqueKeywords(append(meta.tags, meta.keywords...))

	return meta
}

func findBestCollectorContentNode(body *xhtml.Node, baseURL *neturl.URL) *xhtml.Node {
	if preferred := findPreferredCollectorContentNode(body, baseURL); preferred != nil {
		return preferred
	}

	candidates := make([]collectorCandidate, 0, 24)
	walkCollectorNode(body, func(node *xhtml.Node, depth int) bool {
		if node.Type != xhtml.ElementNode {
			return true
		}
		if isCollectorExcludedNode(node) {
			return false
		}
		if !isCollectorCandidateNode(node) {
			return true
		}

		text := collapseCollectorWhitespace(extractCollectorText(node))
		if utf8.RuneCountInString(text) < 60 {
			return true
		}

		score := buildCollectorCandidateScore(node, text)
		if score <= 0 {
			return true
		}

		candidates = append(candidates, collectorCandidate{
			node:    node,
			score:   score,
			textLen: utf8.RuneCountInString(text),
			depth:   depth,
			text:    text,
		})
		return true
	})

	if len(candidates) == 0 {
		return body
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].textLen != candidates[j].textLen {
			return candidates[i].textLen > candidates[j].textLen
		}
		return candidates[i].depth < candidates[j].depth
	})

	return candidates[0].node
}

func findPreferredCollectorContentNode(body *xhtml.Node, baseURL *neturl.URL) *xhtml.Node {
	type preferredCandidate struct {
		node  *xhtml.Node
		score int
		depth int
	}

	candidates := make([]preferredCandidate, 0, 12)
	host := ""
	if baseURL != nil {
		host = strings.ToLower(strings.TrimSpace(baseURL.Hostname()))
	}

	walkCollectorNode(body, func(node *xhtml.Node, depth int) bool {
		if node.Type != xhtml.ElementNode || isCollectorExcludedNode(node) {
			return true
		}

		signals := collectorNodeSignals(node)
		if signals == "" {
			return true
		}

		text := collapseCollectorWhitespace(extractCollectorText(node))
		isAinavContent := strings.Contains(signals, "ainav-tool-content") ||
			strings.Contains(signals, "ainav-article-content") ||
			strings.Contains(signals, "ainav-page-content")
		hasStrongSemanticHint := strings.Contains(signals, "articlebody") ||
			strings.Contains(signals, "role main") ||
			strings.Contains(signals, "main-content") ||
			strings.Contains(signals, "content-main") ||
			strings.Contains(signals, "article-main") ||
			strings.Contains(signals, "article-body") ||
			strings.Contains(signals, "post-body") ||
			strings.Contains(signals, "entry-body")
		if !isAinavContent && !hasStrongSemanticHint && utf8.RuneCountInString(text) < 120 {
			return true
		}

		score := 0
		for _, keyword := range []string{
			"ainav-tool-content",
			"ainav-article-content",
			"ainav-page-content",
			"panel-body single",
			"entry-content",
			"post-content",
			"article-body",
			"post-body",
			"entry-body",
			"content-body",
			"article-main",
			"main-content",
			"content-main",
			"markdown-body",
			"rich-text",
			"richtext",
			"article-content",
			"single-content",
			"single-post",
			"rich_media_content",
			"rich_media_area_primary",
			"detail-content",
			"content-detail",
			"article-detail",
		} {
			if strings.Contains(signals, keyword) {
				score += 240
			}
		}
		if isAinavContent {
			score += 800
		}
		if hasStrongSemanticHint {
			score += 320
		}
		for _, keyword := range []string{"content", "article", "detail", "single", "rich", "post", "entry", "panel-body", "markdown", "main"} {
			if strings.Contains(signals, keyword) {
				score += 40
			}
		}
		for _, keyword := range []string{"comment", "footer", "sidebar", "related", "recommend", "share", "menu", "nav", "toc", "catalog", "author", "bio"} {
			if strings.Contains(signals, keyword) {
				score -= 120
			}
		}
		if strings.Contains(host, "ai-bot.cn") && strings.Contains(signals, "panel-body") && strings.Contains(signals, "single") {
			score += 400
		}
		if score <= 0 {
			return true
		}

		candidates = append(candidates, preferredCandidate{
			node:  node,
			score: score,
			depth: depth,
		})
		return true
	})

	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].depth > candidates[j].depth
	})

	return candidates[0].node
}

func buildCollectorCandidateScore(node *xhtml.Node, text string) int {
	textLen := utf8.RuneCountInString(text)
	score := textLen
	if score > 1800 {
		score = 1800
	}
	signals := collectorNodeSignals(node)
	stats := collectorNodeStats(node)

	switch node.DataAtom {
	case atom.Article:
		score += 240
	case atom.Main:
		score += 200
	case atom.Section:
		score += 120
	case atom.Div:
		score += 80
	}

	if collectorNodeHasMainRole(node) {
		score += 180
	}
	if collectorNodeHasArticleBody(node) {
		score += 260
	}

	for _, keyword := range []string{
		"article", "post", "entry", "content", "detail", "main", "body", "markdown",
		"rich", "single", "panel-body", "article-body", "post-body", "entry-body",
		"content-body", "main-content", "content-main", "article-main", "rich-text", "richtext",
	} {
		if strings.Contains(signals, keyword) {
			score += 100
		}
	}
	for _, keyword := range []string{
		"comment", "comments", "respond", "footer", "copyright", "header", "nav", "menu",
		"breadcrumb", "sidebar", "recommend", "related", "share", "social", "ad", "card-app",
		"url-card", "url-content", "site-item", "post-apd", "toc", "catalog", "table-of-contents",
		"subscribe", "newsletter", "author", "bio",
	} {
		if strings.Contains(signals, keyword) {
			score -= 180
		}
	}

	if strings.Count(text, "。")+strings.Count(text, ".") > 6 {
		score += 60
	}
	if strings.Count(text, "\n") > 4 {
		score += 40
	}
	if stats.headingCount > 0 {
		score += stats.headingCount * 40
	}
	if stats.paragraphCount > 0 {
		score += minCollectorInt(stats.paragraphCount*18, 180)
	}
	if stats.listCount > 0 {
		score += minCollectorInt(stats.listCount*25, 100)
	}
	if stats.imageCount > 0 {
		score += minCollectorInt(stats.imageCount*12, 48)
	}
	if stats.linkCount > 12 {
		score -= (stats.linkCount - 12) * 18
	}
	if stats.elementCount > 180 {
		score -= (stats.elementCount - 180) * 3
	}
	if stats.textLen > 0 && stats.linkTextLen*100/stats.textLen > 45 {
		score -= 220
	}

	return score
}

func isCollectorCandidateNode(node *xhtml.Node) bool {
	switch node.DataAtom {
	case atom.Article, atom.Main, atom.Section, atom.Div:
		return true
	default:
		return false
	}
}

func collectCollectorImages(node *xhtml.Node, baseURL *neturl.URL, imageResolver *collectorImageResolver) []CollectedImage {
	images := make([]CollectedImage, 0, 12)
	seen := make(map[string]struct{})

	walkCollectorNode(node, func(current *xhtml.Node, _ int) bool {
		if current.Type != xhtml.ElementNode {
			return true
		}

		src := collectorNodeImageURL(current, baseURL, imageResolver)
		if src == "" {
			return true
		}
		if shouldSkipCollectorContentImage(current, src, baseURL) {
			return true
		}
		if _, ok := seen[src]; ok {
			return true
		}
		seen[src] = struct{}{}
		alt := collapseCollectorWhitespace(attrValue(current, "alt"))
		if alt == "" {
			alt = collapseCollectorWhitespace(collectorNestedImageAttr(current, "alt"))
		}
		images = append(images, CollectedImage{
			URL: src,
			Alt: alt,
		})

		return len(images) < 20
	})

	return images
}

func collectCollectorOfficialURL(body *xhtml.Node, baseURL *neturl.URL) string {
	if body == nil || baseURL == nil {
		return ""
	}

	type officialCandidate struct {
		url   string
		score int
		depth int
	}

	candidates := make([]officialCandidate, 0, 8)
	walkCollectorNode(body, func(node *xhtml.Node, depth int) bool {
		if node.Type != xhtml.ElementNode || node.DataAtom != atom.A || isCollectorExcludedNode(node) {
			return true
		}

		resolved := resolveCollectorURL(baseURL, attrValue(node, "href"))
		if !collectorLooksLikeNavigableURL(resolved) {
			return true
		}

		score := collectorOfficialLinkScore(node, resolved, baseURL)
		if score <= 0 {
			return true
		}

		candidates = append(candidates, officialCandidate{
			url:   resolved,
			score: score,
			depth: depth,
		})
		return true
	})

	if len(candidates) == 0 {
		return ""
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].depth > candidates[j].depth
	})

	return collectorResolveOfficialURL(candidates[0].url, baseURL)
}

func collectorOfficialLinkScore(node *xhtml.Node, href string, baseURL *neturl.URL) int {
	if node == nil || baseURL == nil {
		return 0
	}

	parsed, err := neturl.Parse(href)
	if err != nil || parsed == nil {
		return 0
	}

	label := strings.ToLower(collapseCollectorWhitespace(
		strings.Join([]string{
			extractCollectorText(node),
			attrValue(node, "title"),
			attrValue(node, "aria-label"),
		}, " "),
	))
	signals := collectorNodeAndAncestorSignals(node, 3)
	score := 0

	for _, keyword := range []string{
		"访问官网", "官网", "官方网站", "前往官网", "立即访问", "立即体验", "打开官网",
		"访问网站", "前往网站", "打开网站", "进入网站", "立即前往", "点击访问", "直达",
		"official", "visit", "website", "site", "open", "launch", "try now",
	} {
		if strings.Contains(label, strings.ToLower(keyword)) {
			score += 320
		}
	}
	for _, keyword := range []string{
		"site-go", "go-url", "visit", "official", "outbound", "external", "btn-arrow", "jump",
	} {
		if strings.Contains(signals, keyword) {
			score += 120
		}
	}
	for _, keyword := range []string{
		"分享", "点赞", "评论", "上一篇", "下一篇", "相关阅读", "相关推荐", "标签", "分类",
		"详情", "阅读", "更多", "share", "like", "comment", "related", "tag", "category",
		"detail", "read more", "previous", "next",
	} {
		if strings.Contains(label, strings.ToLower(keyword)) {
			score -= 260
		}
	}

	rel := strings.ToLower(strings.TrimSpace(attrValue(node, "rel")))
	if collectorLinkHasRel(rel, "nofollow") || collectorLinkHasRel(rel, "sponsored") {
		score += 80
	}
	if strings.EqualFold(strings.TrimSpace(attrValue(node, "target")), "_blank") {
		score += 40
	}
	if collectorLooksLikeRedirectURL(href, baseURL) {
		score += 260
	}
	if !collectorURLHasSameHost(href, baseURL) {
		score += 220
	}
	if collectorLooksLikeInternalDetailURL(href, baseURL) {
		score -= 280
	}

	return score
}

func collectorResolveOfficialURL(candidate string, baseURL *neturl.URL) string {
	if baseURL == nil {
		return ""
	}

	if target := collectorExtractEmbeddedTargetURL(candidate); target != "" {
		return target
	}

	if !collectorURLHasSameHost(candidate, baseURL) {
		normalised, err := normaliseCollectorURL(candidate)
		if err == nil {
			return normalised
		}
		return strings.TrimSpace(candidate)
	}

	if !collectorLooksLikeRedirectURL(candidate, baseURL) {
		return ""
	}

	client := remoteDesktopAPIClient()
	client.Timeout = 15 * time.Second
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	current := candidate
	for attempt := 0; attempt < 5; attempt++ {
		req, err := http.NewRequest(http.MethodGet, current, nil)
		if err != nil {
			break
		}
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) ZQDesktop/0.1.0")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := client.Do(req)
		if err != nil {
			break
		}
		_ = resp.Body.Close()

		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			break
		}
		next := resolveCollectorURL(resp.Request.URL, location)
		if next == "" {
			break
		}

		if target := collectorExtractEmbeddedTargetURL(next); target != "" {
			return target
		}
		if !collectorURLHasSameHost(next, baseURL) {
			normalised, err := normaliseCollectorURL(next)
			if err == nil {
				return normalised
			}
			return next
		}
		current = next
	}

	return ""
}

func collectorLooksLikeNavigableURL(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func collectorLooksLikeRedirectURL(value string, baseURL *neturl.URL) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	if target := collectorExtractEmbeddedTargetURL(trimmed); target != "" {
		return true
	}

	lower := strings.ToLower(trimmed)
	for _, keyword := range []string{"/go/", "/visit/", "/jump/", "/out/", "/link/", "/redirect", "/click/"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	if !collectorURLHasSameHost(trimmed, baseURL) {
		return false
	}

	return strings.Contains(lower, "target=") ||
		strings.Contains(lower, "redirect=") ||
		strings.Contains(lower, "redirect_url=") ||
		strings.Contains(lower, "redirect_uri=") ||
		strings.Contains(lower, "destination=") ||
		strings.Contains(lower, "dest=") ||
		strings.Contains(lower, "url=") ||
		strings.Contains(lower, "to=") ||
		strings.Contains(lower, "out=") ||
		strings.Contains(lower, "link=")
}

func collectorLooksLikeInternalDetailURL(value string, baseURL *neturl.URL) bool {
	if !collectorURLHasSameHost(value, baseURL) {
		return false
	}

	lower := strings.ToLower(value)
	for _, keyword := range []string{"/tool/", "/article/", "/blog", "/category/", "/tag/", "/tags", "/search"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func collectorExtractEmbeddedTargetURL(value string) string {
	parsed, err := neturl.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil {
		return ""
	}

	for _, key := range []string{"target", "url", "u", "to", "dest", "destination", "redirect", "redirect_url", "redirect_uri", "out", "link"} {
		raw := strings.TrimSpace(parsed.Query().Get(key))
		if raw == "" {
			continue
		}
		normalised, err := normaliseCollectorURL(raw)
		if err == nil {
			return normalised
		}
	}

	return ""
}

func collectorURLHasSameHost(value string, baseURL *neturl.URL) bool {
	if baseURL == nil {
		return false
	}

	parsed, err := neturl.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), baseURL.Hostname())
}

func prependCollectorImage(items []CollectedImage, image CollectedImage) []CollectedImage {
	if strings.TrimSpace(image.URL) == "" {
		return items
	}

	filtered := make([]CollectedImage, 0, len(items)+1)
	filtered = append(filtered, image)
	for _, item := range items {
		if item.URL == image.URL {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}
	return filtered
}

func collectCollectorLeadImage(
	contentNode *xhtml.Node,
	body *xhtml.Node,
	baseURL *neturl.URL,
	imageResolver *collectorImageResolver,
	contentImages []CollectedImage,
) *CollectedImage {
	existing := make(map[string]struct{}, len(contentImages))
	for _, item := range contentImages {
		url := strings.TrimSpace(item.URL)
		if url != "" {
			existing[url] = struct{}{}
		}
	}

	if image := findPreferredCollectorLeadImage(contentNode, body, baseURL, imageResolver, existing); image != nil {
		return image
	}

	for current, level := contentNode, 0; current != nil && level < 5; current, level = current.Parent, level+1 {
		for sibling := current.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
			if image := findFirstCollectorMeaningfulImage(sibling, baseURL, imageResolver, existing); image != nil {
				return image
			}
		}
		if current == body {
			break
		}
	}

	return nil
}

func findPreferredCollectorLeadImage(
	contentNode *xhtml.Node,
	body *xhtml.Node,
	baseURL *neturl.URL,
	imageResolver *collectorImageResolver,
	existing map[string]struct{},
) *CollectedImage {
	best := leadImageCandidate{}
	order := 0
	for current, level := contentNode, 0; current != nil && level < 5; current, level = current.Parent, level+1 {
		if candidate := findBestCollectorLeadImageInTree(current, baseURL, imageResolver, existing, level, true, &order); candidate != nil {
			if candidate.score > best.score || (candidate.score == best.score && candidate.order < best.order) {
				best = *candidate
			}
		}
		for sibling := current.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
			if candidate := findBestCollectorLeadImageInTree(sibling, baseURL, imageResolver, existing, level, false, &order); candidate != nil {
				if candidate.score > best.score || (candidate.score == best.score && candidate.order < best.order) {
					best = *candidate
				}
			}
		}
		if current == body {
			break
		}
	}
	return best.image
}

func findBestCollectorLeadImageInTree(
	root *xhtml.Node,
	baseURL *neturl.URL,
	imageResolver *collectorImageResolver,
	existing map[string]struct{},
	ancestorLevel int,
	preferCurrent bool,
	order *int,
) *leadImageCandidate {
	var best *leadImageCandidate
	walkCollectorNode(root, func(node *xhtml.Node, depth int) bool {
		if node.Type != xhtml.ElementNode {
			return true
		}
		score := collectorLeadImageContainerScore(node, depth, ancestorLevel, preferCurrent)
		if score <= 0 {
			return true
		}
		image := findFirstCollectorMeaningfulImage(node, baseURL, imageResolver, existing)
		if image == nil {
			return true
		}
		candidate := &leadImageCandidate{
			image: image,
			score: score,
			order: *order,
		}
		*order = *order + 1
		if best == nil || candidate.score > best.score || (candidate.score == best.score && candidate.order < best.order) {
			best = candidate
		}
		return true
	})
	return best
}

func collectorLeadImageContainerScore(node *xhtml.Node, depth int, ancestorLevel int, preferCurrent bool) int {
	if node == nil || node.Type != xhtml.ElementNode {
		return 0
	}
	classAndID := strings.ToLower(strings.TrimSpace(attrValue(node, "class") + " " + attrValue(node, "id")))
	score := 0
	switch node.DataAtom {
	case atom.Figure:
		score += 120
	case atom.Header:
		score += 80
	case atom.Section:
		score += 60
	case atom.Div:
		score += 40
	case atom.Picture, atom.Img:
		score += 30
	}
	if preferCurrent {
		score += 100
	}
	score += maxCollectorInt(0, 70-ancestorLevel*15)
	score += maxCollectorInt(0, 50-depth*12)

	for _, keyword := range []string{
		"hero", "banner", "cover", "featured", "featured-image", "featured-media",
		"post-thumbnail", "post-thumb", "thumbnail", "thumb", "entry-media",
		"entry-thumbnail", "entry-image", "article-cover", "article-thumb",
		"article-image", "single-cover", "single-thumb", "detail-cover",
		"detail-thumb", "detail-image", "main-image", "main-cover", "head-image",
		"hero-image", "hero-media", "poster", "preview-image", "post-image",
		"top-image", "lead-image", "cover-image", "featured-img", "post-img",
	} {
		if strings.Contains(classAndID, keyword) {
			score += 180
		}
	}
	for _, keyword := range []string{"image", "media", "figure", "screenshot", "preview", "head", "top"} {
		if strings.Contains(classAndID, keyword) {
			score += 45
		}
	}
	for _, keyword := range []string{
		"comment", "comments", "footer", "header-nav", "nav", "menu", "sidebar",
		"related", "recommend", "share", "social", "qrcode", "qr-code", "ad",
		"ads", "advert", "sponsor", "breadcrumb", "pager", "pagination",
	} {
		if strings.Contains(classAndID, keyword) {
			score -= 220
		}
	}

	if score < 150 {
		return 0
	}
	return score
}

func findFirstCollectorMeaningfulImage(
	root *xhtml.Node,
	baseURL *neturl.URL,
	imageResolver *collectorImageResolver,
	existing map[string]struct{},
) *CollectedImage {
	var matched *CollectedImage
	walkCollectorNode(root, func(node *xhtml.Node, _ int) bool {
		if matched != nil || node.Type != xhtml.ElementNode {
			return matched == nil
		}

		src := collectorNodeImageURL(node, baseURL, imageResolver)
		if src == "" || shouldSkipCollectorContentImage(node, src, baseURL) {
			return true
		}
		if _, ok := existing[src]; ok {
			return true
		}

		matched = &CollectedImage{
			URL: src,
			Alt: collapseCollectorWhitespace(firstNonEmpty(
				attrValue(node, "alt"),
				collectorNestedImageAttr(node, "alt"),
			)),
		}
		return false
	})
	return matched
}

func renderCollectorChildren(node *xhtml.Node, baseURL *neturl.URL, imageResolver *collectorImageResolver, depth int) string {
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(renderCollectorNode(child, baseURL, imageResolver, depth))
	}
	return builder.String()
}

func renderCollectorNode(node *xhtml.Node, baseURL *neturl.URL, imageResolver *collectorImageResolver, depth int) string {
	switch node.Type {
	case xhtml.TextNode:
		text := collapseCollectorWhitespace(node.Data)
		if text == "" {
			return ""
		}
		return stdhtml.EscapeString(text)
	case xhtml.ElementNode:
		if isCollectorExcludedNode(node) || depth > 24 {
			return ""
		}

		tag := node.Data
		if !isCollectorAllowedTag(tag) {
			return renderCollectorChildren(node, baseURL, imageResolver, depth+1)
		}

		if tag == "img" || tag == "picture" {
			src := collectorNodeImageURL(node, baseURL, imageResolver)
			if src == "" {
				return ""
			}
			var attrs strings.Builder
			attrs.WriteString(` src="`)
			attrs.WriteString(stdhtml.EscapeString(src))
			attrs.WriteString(`"`)
			alt := collapseCollectorWhitespace(attrValue(node, "alt"))
			if alt == "" {
				alt = collapseCollectorWhitespace(collectorNestedImageAttr(node, "alt"))
			}
			if alt != "" {
				attrs.WriteString(` alt="`)
				attrs.WriteString(stdhtml.EscapeString(alt))
				attrs.WriteString(`"`)
			}
			title := collapseCollectorWhitespace(attrValue(node, "title"))
			if title == "" {
				title = collapseCollectorWhitespace(collectorNestedImageAttr(node, "title"))
			}
			if title != "" {
				attrs.WriteString(` title="`)
				attrs.WriteString(stdhtml.EscapeString(title))
				attrs.WriteString(`"`)
			}
			imageHTML := "<img" + attrs.String() + " />"
			if collectorShouldWrapImageBlock(node) {
				return "<p>" + imageHTML + "</p>"
			}
			return imageHTML
		}

		if tag == "br" || tag == "hr" {
			return "<" + tag + " />"
		}

		var attrs strings.Builder
		if tag == "a" {
			href := resolveCollectorURL(baseURL, attrValue(node, "href"))
			if href != "" {
				attrs.WriteString(` href="`)
				attrs.WriteString(stdhtml.EscapeString(href))
				attrs.WriteString(`" target="_blank" rel="noopener noreferrer"`)
			}
		}

		content := renderCollectorChildren(node, baseURL, imageResolver, depth+1)
		content = strings.TrimSpace(content)
		if content == "" && (tag == "p" || tag == "li" || tag == "blockquote" || strings.HasPrefix(tag, "h")) {
			return ""
		}

		return "<" + tag + attrs.String() + ">" + content + "</" + tag + ">"
	default:
		return ""
	}
}

func isCollectorAllowedTag(tag string) bool {
	switch tag {
	case "p", "br", "img", "a", "strong", "em", "b", "i", "u", "del", "code", "pre", "blockquote",
		"ul", "ol", "li", "table", "thead", "tbody", "tr", "th", "td",
		"h1", "h2", "h3", "h4", "h5", "h6", "hr", "figure", "figcaption", "div", "span":
		return true
	default:
		return false
	}
}

func collectorShouldWrapImageBlock(node *xhtml.Node) bool {
	if node == nil || node.Parent == nil || node.Parent.Type != xhtml.ElementNode {
		return true
	}

	switch node.Parent.DataAtom {
	case atom.A, atom.P, atom.Li, atom.Figure, atom.Figcaption, atom.Td, atom.Th:
		return false
	default:
		return true
	}
}

func containsExcludedKeyword(signals string, keyword string) bool {
	lower := strings.ToLower(signals)
	kw := strings.ToLower(keyword)
	for start := 0; ; {
		idx := strings.Index(lower[start:], kw)
		if idx < 0 {
			return false
		}
		realIdx := start + idx
		end := realIdx + len(kw)
		beforeOK := realIdx == 0 || !isCollectorKeywordContinueChar(rune(lower[realIdx-1]))
		afterOK := end >= len(lower) || !isCollectorKeywordContinueChar(rune(lower[end]))
		if beforeOK && afterOK {
			return true
		}
		start = end
	}
}

func isCollectorKeywordContinueChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
}

func isCollectorExcludedNode(node *xhtml.Node) bool {
	if node.Type != xhtml.ElementNode {
		return false
	}

	if collectorNodeLooksHidden(node) {
		return true
	}

	switch node.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Iframe, atom.Footer, atom.Nav, atom.Form, atom.Button, atom.Input, atom.Textarea, atom.Select, atom.Option, atom.Svg:
		return true
	}

	signals := collectorNodeSignals(node)
	for _, keyword := range []string{
		"comment", "footer", "header", "nav", "menu", "breadcrumb", "sidebar",
		"recommend", "related", "share", "social", "popup", "modal", "ad", "ads",
		"subscribe", "newsletter", "table-of-contents", "toc", "catalog", "cookie",
		"consent", "toolbar", "floating", "floatbar",
	} {
		if containsExcludedKeyword(signals, keyword) {
			return true
		}
	}

	return false
}

func collectorNodeSignals(node *xhtml.Node) string {
	if node == nil {
		return ""
	}

	parts := []string{
		attrValue(node, "class"),
		attrValue(node, "id"),
		attrValue(node, "role"),
		attrValue(node, "itemprop"),
		attrValue(node, "aria-label"),
		attrValue(node, "data-testid"),
		attrValue(node, "data-test"),
		attrValue(node, "data-role"),
		attrValue(node, "data-component"),
		attrValue(node, "data-block"),
	}
	return strings.ToLower(strings.TrimSpace(strings.Join(parts, " ")))
}

func collectorNodeAndAncestorSignals(node *xhtml.Node, maxLevels int) string {
	if node == nil || maxLevels < 0 {
		return ""
	}

	parts := make([]string, 0, maxLevels+1)
	for current, level := node, 0; current != nil && level <= maxLevels; current, level = current.Parent, level+1 {
		signals := collectorNodeSignals(current)
		if signals != "" {
			parts = append(parts, signals)
		}
	}
	return strings.ToLower(strings.TrimSpace(strings.Join(parts, " ")))
}

func collectorNodeLooksHidden(node *xhtml.Node) bool {
	if node == nil {
		return false
	}

	if attrValue(node, "hidden") != "" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(attrValue(node, "aria-hidden")), "true") {
		return true
	}

	style := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(attrValue(node, "style")), " ", ""))
	return strings.Contains(style, "display:none") ||
		strings.Contains(style, "visibility:hidden") ||
		strings.Contains(style, "opacity:0")
}

func collectorNodeHasMainRole(node *xhtml.Node) bool {
	return strings.EqualFold(strings.TrimSpace(attrValue(node, "role")), "main")
}

func collectorNodeHasArticleBody(node *xhtml.Node) bool {
	return strings.EqualFold(strings.TrimSpace(attrValue(node, "itemprop")), "articleBody")
}

func extractCollectorText(node *xhtml.Node) string {
	var builder strings.Builder
	walkCollectorNode(node, func(current *xhtml.Node, _ int) bool {
		switch current.Type {
		case xhtml.TextNode:
			text := strings.TrimSpace(current.Data)
			if text != "" {
				builder.WriteString(text)
				builder.WriteString(" ")
			}
		case xhtml.ElementNode:
			if isCollectorExcludedNode(current) {
				return false
			}
			switch current.DataAtom {
			case atom.Br, atom.P, atom.Div, atom.Article, atom.Section, atom.Li,
				atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
				builder.WriteString("\n")
			}
		}
		return true
	})
	return builder.String()
}

func firstParagraphText(node *xhtml.Node) string {
	var result string
	walkCollectorNode(node, func(current *xhtml.Node, _ int) bool {
		if current.Type != xhtml.ElementNode || current.DataAtom != atom.P {
			return true
		}
		text := collapseCollectorWhitespace(extractCollectorText(current))
		if utf8.RuneCountInString(text) >= 20 {
			result = text
			return false
		}
		return true
	})
	return result
}

func extractHeadingText(node *xhtml.Node) string {
	var result string
	walkCollectorNode(node, func(current *xhtml.Node, _ int) bool {
		if current.Type != xhtml.ElementNode {
			return true
		}
		switch current.DataAtom {
		case atom.H1, atom.H2:
			text := collapseCollectorWhitespace(extractCollectorText(current))
			if text != "" {
				result = text
				return false
			}
		}
		return true
	})
	return result
}

func collapseCollectorWhitespace(text string) string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	return strings.TrimSpace(strings.Join(fields, " "))
}

func truncateCollectorText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func splitCollectorKeywords(text string) []string {
	rawItems := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case ',', '，', ';', '；', '|':
			return true
		default:
			return unicode.IsSpace(r)
		}
	})

	items := make([]string, 0, len(rawItems))
	seen := make(map[string]struct{})
	for _, item := range rawItems {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
		if len(items) >= 12 {
			break
		}
	}

	return items
}

func extractCollectorVisibleTags(contentNode *xhtml.Node, body *xhtml.Node) []string {
	roots := make([]*xhtml.Node, 0, 10)
	seen := make(map[*xhtml.Node]struct{})
	for current, level := contentNode, 0; current != nil && level < 4; current, level = current.Parent, level+1 {
		if _, ok := seen[current]; !ok {
			roots = append(roots, current)
			seen[current] = struct{}{}
		}
		for _, sibling := range []*xhtml.Node{current.PrevSibling, current.NextSibling} {
			if sibling == nil {
				continue
			}
			if _, ok := seen[sibling]; ok {
				continue
			}
			roots = append(roots, sibling)
			seen[sibling] = struct{}{}
		}
		if current == body {
			break
		}
	}
	if body != nil {
		if _, ok := seen[body]; !ok {
			roots = append(roots, body)
		}
	}

	collected := make([]string, 0, 8)
	for _, root := range roots {
		walkCollectorNode(root, func(node *xhtml.Node, depth int) bool {
			if len(collected) >= 24 {
				return false
			}
			if node.Type != xhtml.ElementNode {
				return true
			}
			if collectorTagContainerScore(node, depth) <= 0 && !collectorNodeIsExplicitTagLink(node) {
				return true
			}
			collected = appendCollectorKeywords(collected, collectorTagTextsFromNode(node)...)
			return true
		})
	}
	return collectorUniqueKeywords(collected)
}

func collectorTagContainerScore(node *xhtml.Node, depth int) int {
	if node == nil || node.Type != xhtml.ElementNode {
		return 0
	}
	classAndID := strings.ToLower(strings.TrimSpace(attrValue(node, "class") + " " + attrValue(node, "id")))
	score := 0
	if collectorNodeIsExplicitTagLink(node) {
		score += 300
	}
	for _, keyword := range []string{
		"tag", "tags", "post-tags", "entry-tags", "article-tags", "tag-list",
		"tags-list", "topic", "topics", "topic-list", "label", "labels",
		"keyword", "keywords", "keyword-list", "keyword-cloud", "hashtags", "hash-tags",
		"tagcloud", "tag-cloud", "post-keywords", "entry-keywords", "article-keywords",
	} {
		if strings.Contains(classAndID, keyword) {
			score += 180
		}
	}
	for _, keyword := range []string{
		"breadcrumb", "nav", "menu", "sidebar", "footer", "related", "recommend",
		"share", "social", "author", "profile", "comment", "category", "categories",
	} {
		if strings.Contains(classAndID, keyword) {
			score -= 220
		}
	}
	score += maxCollectorInt(0, 36-depth*8)
	if score < 120 {
		return 0
	}
	return score
}

func collectorNodeIsExplicitTagLink(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode || node.DataAtom != atom.A {
		return false
	}
	rel := strings.ToLower(strings.TrimSpace(attrValue(node, "rel")))
	href := strings.ToLower(strings.TrimSpace(attrValue(node, "href")))
	classAndID := strings.ToLower(strings.TrimSpace(attrValue(node, "class") + " " + attrValue(node, "id")))
	return collectorLinkHasRel(rel, "tag") ||
		strings.Contains(href, "/tag/") ||
		strings.Contains(href, "/tags/") ||
		strings.Contains(href, "/topic/") ||
		strings.Contains(href, "/topics/") ||
		strings.Contains(href, "/label/") ||
		strings.Contains(href, "/labels/") ||
		strings.Contains(href, "/keyword/") ||
		strings.Contains(href, "/keywords/") ||
		strings.Contains(classAndID, "tag") ||
		strings.Contains(classAndID, "topic") ||
		strings.Contains(classAndID, "label") ||
		strings.Contains(classAndID, "keyword")
}

func collectorTagTextsFromNode(node *xhtml.Node) []string {
	if node == nil || node.Type != xhtml.ElementNode {
		return nil
	}
	collected := make([]string, 0, 6)
	addText := func(text string) {
		value := collapseCollectorWhitespace(strings.Trim(text, "#"))
		length := utf8.RuneCountInString(value)
		if value == "" || length <= 1 || length > 30 {
			return
		}
		if strings.ContainsAny(value, "\n\r\t") {
			return
		}
		lower := strings.ToLower(value)
		for _, keyword := range []string{"上一篇", "下一篇", "相关推荐", "相关阅读", "分享", "评论", "作者"} {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return
			}
		}
		collected = appendCollectorKeywords(collected, value)
	}
	if collectorNodeIsExplicitTagLink(node) {
		addText(extractCollectorText(node))
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.TextNode {
			addText(child.Data)
			continue
		}
		if child.Type != xhtml.ElementNode {
			continue
		}
		switch child.DataAtom {
		case atom.A, atom.Span, atom.Button, atom.Li:
			addText(extractCollectorText(child))
		}
	}
	return collected
}

func appendCollectorKeywords(items []string, values ...string) []string {
	merged := append([]string{}, items...)
	for _, value := range values {
		merged = append(merged, splitCollectorKeywords(value)...)
	}
	return collectorUniqueKeywords(merged)
}

func collectorUniqueKeywords(values []string, excluded ...string) []string {
	items := make([]string, 0, len(values))
	excludedSet := buildCollectorKeywordExclusionSet(excluded...)
	seen := make(map[string]struct{})
	for _, value := range values {
		trimmed := normalizeCollectorKeyword(value)
		if !isMeaningfulCollectorKeyword(trimmed, excludedSet) {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, trimmed)
		if len(items) >= 12 {
			break
		}
	}
	return items
}

func buildCollectorFallbackKeywords(values ...string) []string {
	replacer := strings.NewReplacer(
		"｜", "|",
		"|", "|",
		"—", "|",
		"–", "|",
		"-", "|",
		"_", "|",
		"/", "|",
		"\\", "|",
		"，", ",",
		"、", ",",
		"；", ",",
		";", ",",
		"：", ",",
		":", ",",
		"（", " ",
		"）", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"【", " ",
		"】", " ",
		"·", " ",
	)

	candidates := make([]string, 0, len(values)*2)
	for index, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if index == 0 && utf8.RuneCountInString(trimmed) <= 24 {
			candidates = append(candidates, trimmed)
		}
		normalised := replacer.Replace(trimmed)
		candidates = append(candidates, splitCollectorKeywords(normalised)...)
	}

	return collectorUniqueKeywords(candidates)
}

func normalizeCollectorKeyword(value string) string {
	trimmed := collapseCollectorWhitespace(stdhtml.UnescapeString(value))
	trimmed = strings.Trim(trimmed, " \t\r\n,.;:!?|/\\-_()[]{}<>\"'`~@#$%^&*+=，。；：！？、（）【】《》")
	return collapseCollectorWhitespace(trimmed)
}

func buildCollectorKeywordExclusionSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, value := range values {
		normalised := strings.ToLower(normalizeCollectorKeyword(value))
		if normalised == "" {
			continue
		}
		set[normalised] = struct{}{}
		for _, token := range strings.FieldsFunc(normalised, func(r rune) bool {
			switch r {
			case '.', '-', '_', '/', '\\', '|', ',', '，', ';', '；', ':', '：':
				return true
			default:
				return unicode.IsSpace(r)
			}
		}) {
			if utf8.RuneCountInString(token) >= 2 {
				set[token] = struct{}{}
			}
		}
	}
	return set
}

func isMeaningfulCollectorKeyword(value string, excludedSet map[string]struct{}) bool {
	trimmed := normalizeCollectorKeyword(value)
	if trimmed == "" {
		return false
	}
	length := utf8.RuneCountInString(trimmed)
	if length < 2 || length > 24 {
		return false
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	if _, excluded := excludedSet[lower]; excluded {
		return false
	}
	if strings.HasPrefix(lower, "www.") || strings.Contains(lower, ".com") || strings.Contains(lower, ".cn") || strings.Contains(lower, ".net") {
		return false
	}
	if isCollectorNumericKeyword(lower) || isCollectorDateKeyword(lower) {
		return false
	}

	switch lower {
	case "首页", "官网", "官方网站", "官方", "网站", "平台", "详情", "文章", "工具", "新闻",
		"最新", "更多", "推荐", "精选", "热门", "下载", "登录", "注册", "阅读", "查看", "发布",
		"home", "index", "official", "website", "site", "platform", "detail", "details",
		"article", "articles", "tool", "tools", "news", "blog", "blogs", "read", "more",
		"latest", "update", "updates", "login", "sign", "signup", "register", "download":
		return false
	}

	hasLetterOrDigit := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasLetterOrDigit = true
			break
		}
	}
	return hasLetterOrDigit
}

func isCollectorNumericKeyword(value string) bool {
	hasDigit := false
	for _, r := range value {
		if unicode.IsDigit(r) {
			hasDigit = true
			continue
		}
		if r == '.' || r == '-' || r == '_' || r == '+' {
			continue
		}
		return false
	}
	return hasDigit
}

func isCollectorDateKeyword(value string) bool {
	if len(value) < 4 || len(value) > 10 {
		return false
	}

	digits := 0
	for _, r := range value {
		if unicode.IsDigit(r) {
			digits++
			continue
		}
		switch r {
		case '-', '/', '.', '_':
		default:
			return false
		}
	}
	return digits >= 4
}

func extractCollectorJSONLDMeta(raw string, meta *collectorMeta, baseURL *neturl.URL) {
	content := strings.TrimSpace(raw)
	if content == "" || meta == nil {
		return
	}

	var payload any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return
	}

	walkCollectorJSONLD(payload, meta, baseURL)
}

func walkCollectorJSONLD(value any, meta *collectorMeta, baseURL *neturl.URL) {
	switch current := value.(type) {
	case []any:
		for _, item := range current {
			walkCollectorJSONLD(item, meta, baseURL)
		}
	case map[string]any:
		if meta.title == "" {
			meta.title = firstNonEmpty(meta.title, collectorJSONLDString(current, "headline"), collectorJSONLDString(current, "name"))
		}
		if meta.description == "" {
			meta.description = firstNonEmpty(meta.description, collectorJSONLDString(current, "description"))
		}
		if meta.siteName == "" {
			if publisher, ok := current["publisher"].(map[string]any); ok {
				meta.siteName = firstNonEmpty(meta.siteName, collectorJSONLDString(publisher, "name"))
			}
		}
		if meta.publishedAt == "" {
			meta.publishedAt = firstNonEmpty(meta.publishedAt, collectorJSONLDString(current, "datePublished"), collectorJSONLDString(current, "dateCreated"))
		}
		if meta.ogImage == "" {
			meta.ogImage = firstNonEmpty(
				meta.ogImage,
				collectorJSONLDImageURL(current["image"], baseURL),
				collectorJSONLDImageURL(current["thumbnailUrl"], baseURL),
			)
		}
		if meta.icon == "" {
			meta.icon = firstNonEmpty(
				meta.icon,
				collectorJSONLDImageURL(current["logo"], baseURL),
			)
			if meta.icon == "" {
				if publisher, ok := current["publisher"].(map[string]any); ok {
					meta.icon = firstNonEmpty(meta.icon, collectorJSONLDImageURL(publisher["logo"], baseURL))
				}
			}
		}

		meta.keywords = appendCollectorKeywords(
			meta.keywords,
			collectorJSONLDKeywords(current["keywords"])...,
		)
		meta.tags = appendCollectorKeywords(
			meta.tags,
			collectorJSONLDKeywords(current["articleSection"])...,
		)
		meta.tags = appendCollectorKeywords(
			meta.tags,
			collectorJSONLDKeywords(current["about"])...,
		)
		meta.tags = appendCollectorKeywords(
			meta.tags,
			collectorJSONLDKeywords(current["genre"])...,
		)
		meta.tags = appendCollectorKeywords(
			meta.tags,
			collectorJSONLDKeywords(current["knowsAbout"])...,
		)

		for _, child := range current {
			walkCollectorJSONLD(child, meta, baseURL)
		}
	}
}

func collectorJSONLDString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch item := value.(type) {
	case string:
		return collapseCollectorWhitespace(item)
	case []any:
		for _, child := range item {
			if text, ok := child.(string); ok {
				return collapseCollectorWhitespace(text)
			}
		}
	}
	return ""
}

func collectorJSONLDKeywords(value any) []string {
	switch item := value.(type) {
	case string:
		return splitCollectorKeywords(item)
	case []any:
		values := make([]string, 0, len(item))
		for _, child := range item {
			switch value := child.(type) {
			case string:
				values = append(values, value)
			case map[string]any:
				values = append(values,
					collectorJSONLDString(value, "name"),
					collectorJSONLDString(value, "title"),
					collectorJSONLDString(value, "@value"),
				)
			}
		}
		return collectorUniqueKeywords(values)
	case map[string]any:
		values := []string{
			collectorJSONLDString(item, "name"),
			collectorJSONLDString(item, "title"),
			collectorJSONLDString(item, "@value"),
		}
		if itemList, ok := item["itemListElement"]; ok {
			values = append(values, collectorJSONLDKeywords(itemList)...)
		}
		if about, ok := item["about"]; ok {
			values = append(values, collectorJSONLDKeywords(about)...)
		}
		if keywords, ok := item["keywords"]; ok {
			values = append(values, collectorJSONLDKeywords(keywords)...)
		}
		return collectorUniqueKeywords(values)
	default:
		return nil
	}
}

func collectorJSONLDImageURL(value any, baseURL *neturl.URL) string {
	switch item := value.(type) {
	case string:
		return resolveCollectorURL(baseURL, item)
	case []any:
		for _, child := range item {
			if matched := collectorJSONLDImageURL(child, baseURL); matched != "" {
				return matched
			}
		}
	case map[string]any:
		return firstNonEmpty(
			collectorJSONLDImageURL(item["url"], baseURL),
			collectorJSONLDImageURL(item["contentUrl"], baseURL),
			collectorJSONLDImageURL(item["@id"], baseURL),
			collectorJSONLDImageURL(item["image"], baseURL),
			collectorJSONLDImageURL(item["thumbnailUrl"], baseURL),
			collectorJSONLDImageURL(item["logo"], baseURL),
		)
	}
	return ""
}

func resolveCollectorURL(baseURL *neturl.URL, raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || baseURL == nil {
		return ""
	}
	if strings.HasPrefix(value, "//") {
		return baseURL.Scheme + ":" + value
	}

	parsed, err := neturl.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		return parsed.String()
	}

	return baseURL.ResolveReference(parsed).String()
}

func newCollectorImageResolver(pageURL *neturl.URL) *collectorImageResolver {
	client := remoteDesktopAPIClient()
	client.Timeout = 20 * time.Second
	return &collectorImageResolver{
		pageURL: pageURL,
		client:  client,
		cache:   make(map[string]string),
	}
}

func collectorNodeImageURL(node *xhtml.Node, baseURL *neturl.URL, imageResolver *collectorImageResolver) string {
	raw := firstNonEmpty(
		collectorDirectImageCandidate(node),
		collectorBackgroundImageCandidate(node),
		collectorNoscriptImageCandidate(node),
	)
	if raw == "" && node != nil && node.Parent != nil && node.Parent.Type == xhtml.ElementNode && node.Parent.DataAtom == atom.Picture {
		raw = collectorPictureImageCandidate(node.Parent)
	}
	if raw == "" && node != nil && node.Type == xhtml.ElementNode && node.DataAtom == atom.Picture {
		raw = collectorPictureImageCandidate(node)
	}
	if imageResolver == nil {
		return resolveCollectorURL(baseURL, raw)
	}
	return imageResolver.resolve(raw)
}

func collectorDirectImageCandidate(node *xhtml.Node) string {
	if node == nil || node.Type != xhtml.ElementNode {
		return ""
	}
	return normalizeCollectorImageCandidate(firstNonEmpty(
		attrValue(node, "data-src"),
		attrValue(node, "data-original"),
		attrValue(node, "data-actualsrc"),
		attrValue(node, "data-lazy-src"),
		attrValue(node, "data-srcset"),
		attrValue(node, "data-original-srcset"),
		attrValue(node, "data-lazy-srcset"),
		attrValue(node, "data-fallback-src"),
		attrValue(node, "data-thumb"),
		attrValue(node, "data-thumbnail"),
		attrValue(node, "data-cover"),
		attrValue(node, "data-bg"),
		attrValue(node, "data-bg-src"),
		attrValue(node, "data-background"),
		attrValue(node, "data-url"),
		attrValue(node, "data-image"),
		attrValue(node, "data-echo"),
		attrValue(node, "data-ks-lazyload"),
		attrValue(node, "data-defer-src"),
		attrValue(node, "data-lazyload"),
		attrValue(node, "data-orig-file"),
		attrValue(node, "data-large-file"),
		attrValue(node, "original"),
		attrValue(node, "srcset"),
		attrValue(node, "src"),
	))
}

func collectorBackgroundImageCandidate(node *xhtml.Node) string {
	if node == nil || node.Type != xhtml.ElementNode {
		return ""
	}
	return normalizeCollectorImageCandidate(firstNonEmpty(
		extractCollectorBackgroundImage(attrValue(node, "style")),
		attrValue(node, "data-background"),
		attrValue(node, "data-bg"),
		attrValue(node, "data-bg-src"),
	))
}

func extractCollectorBackgroundImage(style string) string {
	value := strings.TrimSpace(style)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	start := strings.Index(lower, "url(")
	if start < 0 {
		return ""
	}
	segment := value[start+4:]
	end := strings.Index(segment, ")")
	if end < 0 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(segment[:end]), `"'`)
}

func collectorPictureImageCandidate(node *xhtml.Node) string {
	if node == nil || node.Type != xhtml.ElementNode {
		return ""
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode {
			continue
		}
		raw := collectorDirectImageCandidate(child)
		if raw != "" {
			return raw
		}
	}
	return ""
}

func normalizeCollectorImageCandidate(raw string) string {
	value := strings.Trim(strings.TrimSpace(raw), `"'`)
	if value == "" {
		return ""
	}
	bestURL := ""
	bestScore := -1
	for _, item := range strings.Split(value, ",") {
		candidate := strings.Trim(strings.TrimSpace(item), `"'`)
		if candidate == "" {
			continue
		}
		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			continue
		}
		url := strings.Trim(strings.TrimSpace(fields[0]), `"'`)
		if url == "" || strings.HasPrefix(strings.ToLower(url), "javascript:") {
			continue
		}
		score := collectorImageCandidateScore(fields)
		if score > bestScore {
			bestScore = score
			bestURL = url
		}
	}
	return bestURL
}

func collectorImageCandidateScore(fields []string) int {
	if len(fields) <= 1 {
		return 1
	}
	descriptor := strings.ToLower(strings.TrimSpace(fields[1]))
	if strings.HasSuffix(descriptor, "w") {
		widthText := strings.TrimSpace(strings.TrimSuffix(descriptor, "w"))
		width := 0
		for _, r := range widthText {
			if !unicode.IsDigit(r) {
				return 1
			}
			width = width*10 + int(r-'0')
		}
		if width > 0 {
			return width
		}
	}
	if strings.HasSuffix(descriptor, "x") {
		scaleText := strings.TrimSpace(strings.TrimSuffix(descriptor, "x"))
		parts := strings.SplitN(scaleText, ".", 2)
		value := 0
		for _, part := range parts {
			if part == "" {
				continue
			}
			for _, r := range part {
				if !unicode.IsDigit(r) {
					return 1
				}
				value = value*10 + int(r-'0')
			}
		}
		if value > 0 {
			return value * 1000
		}
	}
	return 1
}

func collectorNoscriptImageCandidate(node *xhtml.Node) string {
	if node == nil || node.Type != xhtml.ElementNode || node.DataAtom != atom.Noscript {
		return ""
	}
	rawHTML := collectorRawInnerHTML(node)
	if strings.TrimSpace(rawHTML) == "" {
		return ""
	}
	root, err := xhtml.Parse(strings.NewReader("<html><body>" + rawHTML + "</body></html>"))
	if err != nil {
		return ""
	}
	body := findHTMLNode(root, atom.Body)
	if body == nil {
		return ""
	}
	var matched string
	walkCollectorNode(body, func(current *xhtml.Node, _ int) bool {
		if matched != "" || current.Type != xhtml.ElementNode {
			return matched == ""
		}
		matched = firstNonEmpty(
			collectorDirectImageCandidate(current),
			collectorBackgroundImageCandidate(current),
		)
		return matched == ""
	})
	return matched
}

func collectorRawInnerHTML(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := xhtml.Render(&builder, child); err != nil {
			return ""
		}
	}
	return builder.String()
}

func collectorNestedImageAttr(node *xhtml.Node, key string) string {
	if node == nil || node.Type != xhtml.ElementNode {
		return ""
	}
	if value := attrValue(node, key); strings.TrimSpace(value) != "" {
		return value
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode {
			continue
		}
		if value := attrValue(child, key); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func shouldSkipCollectorContentImage(node *xhtml.Node, src string, baseURL *neturl.URL) bool {
	if strings.TrimSpace(src) == "" {
		return true
	}

	lowerSrc := strings.ToLower(strings.TrimSpace(src))
	if strings.HasPrefix(lowerSrc, "data:image/gif") {
		return true
	}
	if strings.Contains(lowerSrc, "/wp-content/themes/onenav/images/fx/shape-") {
		return true
	}
	if strings.HasSuffix(lowerSrc, "/wp-content/themes/onenav/images/t.png") {
		return true
	}

	parsed, err := neturl.Parse(src)
	if err == nil {
		path := strings.ToLower(parsed.Path)
		for _, keyword := range []string{"qrcode", "qr-code", "sprite"} {
			if strings.Contains(path, keyword) {
				return true
			}
		}
	}

	classAndID := strings.ToLower(strings.TrimSpace(attrValue(node, "class") + " " + attrValue(node, "id")))
	for _, keyword := range []string{"header", "footer", "nav", "qrcode", "qr-code"} {
		if strings.Contains(classAndID, keyword) {
			return true
		}
	}

	width := strings.TrimSpace(attrValue(node, "width"))
	height := strings.TrimSpace(attrValue(node, "height"))
	if width != "" && height != "" && (width == "1" || width == "2") && (height == "1" || height == "2") {
		return true
	}

	return false
}

func (r *collectorImageResolver) resolve(raw string) string {
	resolved := resolveCollectorURL(r.pageURL, raw)
	if resolved == "" || !shouldInlineCollectorImage(r.pageURL, resolved) {
		return resolved
	}

	inlined, err := r.inlineDataURL(resolved)
	if err != nil {
		return resolved
	}
	return inlined
}

func (r *collectorImageResolver) inlineDataURL(imageURL string) (string, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(imageURL)), "data:") {
		return imageURL, nil
	}
	if cached, ok := r.cache[imageURL]; ok {
		return cached, nil
	}

	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) ZQDesktop/0.1.0")
	req.Header.Set("Cache-Control", "no-cache")
	if r.pageURL != nil {
		req.Header.Set("Referer", r.pageURL.String())
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("fetch image failed: %d", resp.StatusCode)
	}

	const maxCollectorImageBytes = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCollectorImageBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) == 0 {
		return "", fmt.Errorf("empty image body")
	}
	if len(body) > maxCollectorImageBytes {
		return "", fmt.Errorf("image too large")
	}

	mimeType := collectorImageMimeType(resp.Header.Get("Content-Type"), body)
	dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(body)
	r.cache[imageURL] = dataURL
	return dataURL, nil
}

func shouldInlineCollectorImage(pageURL *neturl.URL, imageURL string) bool {
	if pageURL == nil || !isWeChatCollectorHost(pageURL.Hostname()) {
		return false
	}
	parsed, err := neturl.Parse(strings.TrimSpace(imageURL))
	if err != nil {
		return false
	}
	if parsed.Scheme == "data" {
		return false
	}
	return isWeChatCollectorHost(parsed.Hostname())
}

func isWeChatCollectorHost(host string) bool {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == "" {
		return false
	}
	return strings.Contains(value, "weixin.qq.com") ||
		strings.Contains(value, "mmbiz.qpic.cn") ||
		strings.Contains(value, "mmbiz.qlogo.cn") ||
		strings.Contains(value, "mmbizpic.cn") ||
		strings.Contains(value, "res.wx.qq.com")
}

func collectorImageMimeType(contentType string, body []byte) string {
	if mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType)); err == nil && strings.HasPrefix(mediaType, "image/") {
		return mediaType
	}
	detected := http.DetectContentType(body)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "image/jpeg"
}

func findHTMLNode(root *xhtml.Node, target atom.Atom) *xhtml.Node {
	var result *xhtml.Node
	walkCollectorNode(root, func(node *xhtml.Node, _ int) bool {
		if node.Type == xhtml.ElementNode && node.DataAtom == target {
			result = node
			return false
		}
		return true
	})
	return result
}

func walkCollectorNode(node *xhtml.Node, visit func(*xhtml.Node, int) bool) {
	var walker func(*xhtml.Node, int) bool
	walker = func(current *xhtml.Node, depth int) bool {
		if current == nil {
			return true
		}
		if !visit(current, depth) {
			return false
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			if !walker(child, depth+1) {
				return false
			}
		}
		return true
	}

	_ = walker(node, 0)
}

func attrValue(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func collectorLinkHasRel(rel string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, token := range strings.Fields(strings.ToLower(strings.TrimSpace(rel))) {
		if token == target {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type collectorStats struct {
	elementCount   int
	linkCount      int
	imageCount     int
	listCount      int
	headingCount   int
	paragraphCount int
	textLen        int
	linkTextLen    int
}

func collectorNodeStats(node *xhtml.Node) collectorStats {
	var stats collectorStats
	walkCollectorNode(node, func(current *xhtml.Node, _ int) bool {
		switch current.Type {
		case xhtml.TextNode:
			textLen := utf8.RuneCountInString(collapseCollectorWhitespace(current.Data))
			stats.textLen += textLen
			if collectorNodeHasAncestor(current, atom.A) {
				stats.linkTextLen += textLen
			}
		case xhtml.ElementNode:
			stats.elementCount++
			switch current.DataAtom {
			case atom.A:
				stats.linkCount++
			case atom.Img, atom.Picture:
				stats.imageCount++
			case atom.Ul, atom.Ol:
				stats.listCount++
			case atom.P:
				stats.paragraphCount++
			case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
				stats.headingCount++
			}
		}
		return true
	})
	return stats
}

func collectorNodeHasAncestor(node *xhtml.Node, target atom.Atom) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type == xhtml.ElementNode && current.DataAtom == target {
			return true
		}
	}
	return false
}

func minCollectorInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxCollectorInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func scanSiteLinks(input ScanSiteLinksInput) (*ScanSiteLinksResult, error) {
	pageURL, err := normaliseCollectorURL(input.URL)
	if err != nil {
		return nil, err
	}

	maxLinks := input.MaxLinks
	if maxLinks < 1 {
		maxLinks = 50
	}
	if maxLinks > 500 {
		maxLinks = 500
	}

	baseURL, parseErr := neturl.Parse(pageURL)
	if parseErr != nil {
		return nil, fmt.Errorf("解析页面地址失败：%w", parseErr)
	}

	scanResultHost := baseURL.Hostname()
	seen := make(map[string]bool)
	var links []SiteLinkItem
	pageHTMLCount := 0

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建扫描请求失败：%w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) ZQDesktop/0.1.0")
	req.Header.Set("Cache-Control", "no-cache")

	client := remoteDesktopAPIClient()
	client.Timeout = 25 * time.Second

	resp, doErr := client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("%s", describeRemoteRequestError(doErr))
	}
	defer resp.Body.Close()

	meta := collectorMeta{}
	var root *xhtml.Node
	title := baseURL.String()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
		if contentType == "" || strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml+xml") {
			bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
			if readErr == nil && len(bodyBytes) > 0 {
				parsedRoot, parseErr := xhtml.Parse(strings.NewReader(string(bodyBytes)))
				if parseErr == nil {
					root = parsedRoot
					meta = extractCollectorMeta(root, baseURL)
					title = firstNonEmpty(meta.title, extractHeadingText(findHTMLNode(root, atom.Body)), baseURL.String())

					var walkFn func(*xhtml.Node)
					walkFn = func(node *xhtml.Node) {
						if len(links) >= maxLinks {
							return
						}
						if node.Type == xhtml.ElementNode && node.DataAtom == atom.A {
							href := strings.TrimSpace(attrValue(node, "href"))
							if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
								return
							}

							resolved := resolveCollectorURL(baseURL, href)
							if resolved == "" || !collectorLooksLikeNavigableURL(resolved) {
								return
							}

							resolvedURL, urlErr := neturl.Parse(resolved)
							if urlErr != nil || resolvedURL == nil {
								return
							}

							if resolvedURL.Hostname() != scanResultHost {
								return
							}

							if collectorLooksLikeRedirectURL(resolved, baseURL) {
								return
							}

							cleanURL := resolvedURL.Scheme + "://" + resolvedURL.Host + resolvedURL.Path
							if seen[cleanURL] {
								return
							}

							ext := strings.ToLower(filepath.Ext(resolvedURL.Path))
							if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".svg" || ext == ".webp" || ext == ".ico" || ext == ".css" || ext == ".js" || ext == ".json" || ext == ".xml" || ext == ".rss" || ext == ".atom" || ext == ".pdf" || ext == ".zip" || ext == ".rar" {
								return
							}
							if resolvedURL.Path == "" || resolvedURL.Path == "/" {
								return
							}

							if collectorNodeHasAncestor(node, atom.Nav) || collectorNodeHasAncestor(node, atom.Header) || collectorNodeHasAncestor(node, atom.Aside) {
								return
							}
							signals := collectorNodeAndAncestorSignals(node, 4)
							lowerSignals := strings.ToLower(signals)
							if strings.Contains(lowerSignals, "nav") || strings.Contains(lowerSignals, "menu") || strings.Contains(lowerSignals, "tab") || strings.Contains(lowerSignals, "navbar") || strings.Contains(lowerSignals, "breadcrumb") || strings.Contains(lowerSignals, "pagination") || strings.Contains(lowerSignals, "footer") || strings.Contains(lowerSignals, "sidebar") || strings.Contains(lowerSignals, "aside") || strings.Contains(lowerSignals, "widget") || strings.Contains(lowerSignals, "category-list") || strings.Contains(lowerSignals, "archive") {
								return
							}

							linkText := strings.TrimSpace(extractCollectorText(node))
							linkTitle := firstNonEmpty(linkText, strings.TrimSpace(attrValue(node, "title")))

							seen[cleanURL] = true
							links = append(links, SiteLinkItem{
								URL:    cleanURL,
								Title:  linkTitle,
								Source: "html",
							})
							return
						}

						for child := node.FirstChild; child != nil; child = child.NextSibling {
							if len(links) >= maxLinks {
								return
							}
							walkFn(child)
						}
					}

					walkFn(root)
				}
			}
		}
	}

	pageHTMLCount = len(links)

	var sitemapSources []string
	sitemapURLCount := 0

	if input.ScanSitemap && len(links) < maxLinks {
		sitemapURLs := collectorFetchSitemapURLs(baseURL, client)
		sitemapSources = sitemapURLs
		for _, sitemapURL := range sitemapURLs {
			if len(links) >= maxLinks {
				break
			}
			sitemapLinks := collectorParseSitemap(sitemapURL, client, scanResultHost, maxLinks-len(links))
			for _, item := range sitemapLinks {
				if seen[item.URL] {
					continue
				}
				seen[item.URL] = true
				item.Source = "sitemap"
				links = append(links, item)
				if len(links) >= maxLinks {
					break
				}
			}
		}
		sitemapURLCount = len(links) - pageHTMLCount
	}

	if len(input.FilterRules) > 0 && len(links) > 0 {
		var filtered []SiteLinkItem
		for _, item := range links {
			if applyScanFilterRules(item, input.FilterRules) {
				filtered = append(filtered, item)
			}
		}
		links = filtered
	}

	if links == nil {
		links = make([]SiteLinkItem, 0)
	}

	finalURL := pageURL
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	return &ScanSiteLinksResult{
		RequestedURL:    pageURL,
		FinalURL:        finalURL,
		Host:            scanResultHost,
		SiteName:        meta.siteName,
		Title:           title,
		Links:           links,
		PageHTMLCount:   pageHTMLCount,
		SitemapURLCount: sitemapURLCount,
		SitemapSources:  sitemapSources,
		ScannedAt:       time.Now().Format(time.RFC3339),
	}, nil
}

type sitemapIndex struct {
	XMLName xml.Name       `xml:"sitemapindex"`
	Sitemap []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	URL     []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func collectorFetchSitemapURLs(baseURL *neturl.URL, client *http.Client) []string {
	robotsURL := baseURL.Scheme + "://" + baseURL.Host + "/robots.txt"
	req, err := http.NewRequest(http.MethodGet, robotsURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "ZQDesktop/0.1.0")

	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<19))
	if err != nil || len(bodyBytes) == 0 {
		return nil
	}

	var sitemapURLs []string
	lines := strings.Split(string(bodyBytes), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "sitemap:") {
			candidate := strings.TrimSpace(trimmed[len("sitemap:"):])
			if resolved := resolveCollectorURL(baseURL, candidate); resolved != "" && collectorLooksLikeNavigableURL(resolved) {
				sitemapURLs = append(sitemapURLs, resolved)
			}
		}
	}

	if len(sitemapURLs) == 0 {
		candidate := baseURL.Scheme + "://" + baseURL.Host + "/sitemap.xml"
		sitemapURLs = append(sitemapURLs, candidate)
	}

	return sitemapURLs
}

func collectorParseSitemap(sitemapURL string, client *http.Client, host string, max int) []SiteLinkItem {
	req, err := http.NewRequest(http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "ZQDesktop/0.1.0")

	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil || len(bodyBytes) == 0 {
		return nil
	}

	var index sitemapIndex
	if xml.Unmarshal(bodyBytes, &index) == nil && len(index.Sitemap) > 0 {
		var allLinks []SiteLinkItem
		for _, entry := range index.Sitemap {
			if len(allLinks) >= max {
				break
			}
			parsed, urlErr := neturl.Parse(entry.Loc)
			if urlErr != nil || parsed == nil || parsed.Hostname() != host {
				continue
			}
			subLinks := collectorParseSitemap(entry.Loc, client, host, max-len(allLinks))
			allLinks = append(allLinks, subLinks...)
		}
		return allLinks
	}

	var urlset urlSet
	if xml.Unmarshal(bodyBytes, &urlset) == nil && len(urlset.URL) > 0 {
		var links []SiteLinkItem
		for _, entry := range urlset.URL {
			if len(links) >= max {
				break
			}
			cleanURL := strings.TrimSpace(entry.Loc)
			if cleanURL == "" || !collectorLooksLikeNavigableURL(cleanURL) {
				continue
			}
			parsed, urlErr := neturl.Parse(cleanURL)
			if urlErr != nil || parsed == nil || parsed.Hostname() != host {
				continue
			}
			links = append(links, SiteLinkItem{
				URL:   cleanURL,
				Title: "",
			})
		}
		return links
	}

	return nil
}

func applyScanFilterRules(item SiteLinkItem, rules []ScanFilterRule) bool {
	target := ""
	for _, rule := range rules {
		switch rule.Field {
		case "url":
			target = item.URL
		case "title":
			target = item.Title
		default:
			target = strings.ToLower(item.URL + " " + item.Title)
		}

		lowerTarget := strings.ToLower(target)
		lowerValue := strings.ToLower(rule.Value)

		switch rule.Operator {
		case "contains":
			if !strings.Contains(lowerTarget, lowerValue) {
				return false
			}
		case "not_contains":
			if strings.Contains(lowerTarget, lowerValue) {
				return false
			}
		case "prefix":
			if !strings.HasPrefix(lowerTarget, lowerValue) {
				return false
			}
		case "suffix":
			if !strings.HasSuffix(lowerTarget, lowerValue) {
				return false
			}
		case "regex":
			matched, _ := filepath.Match(rule.Value, target)
			if !matched {
				return false
			}
		case "path_prefix":
			parsed, err := neturl.Parse(item.URL)
			if err != nil || !strings.HasPrefix(strings.ToLower(parsed.Path), lowerValue) {
				return false
			}
		case "path_contains":
			parsed, err := neturl.Parse(item.URL)
			if err != nil || !strings.Contains(strings.ToLower(parsed.Path), lowerValue) {
				return false
			}
		default:
			if !strings.Contains(lowerTarget, lowerValue) {
				return false
			}
		}
	}
	return true
}
