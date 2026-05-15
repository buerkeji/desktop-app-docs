package main

import (
	"fmt"
	stdhtml "html"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func collectWeChatPage(input CollectWebPageInput) (*CollectedWebPageResult, error) {
	pageURL, err := normaliseCollectorURL(input.URL)
	if err != nil {
		return nil, err
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
		return collectWeChatPageFromHTML(pageURL, renderedURL, []byte(renderedPreview.HTML), renderedPreview.HTML)
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
					return collectWeChatPageFromHTML(pageURL, renderedURL, []byte(renderedPreview.HTML), renderedPreview.HTML)
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

	return collectWeChatPageFromHTML(pageURL, resp.Request.URL, bodyBytes, "")
}

func collectWeChatPageFromHTML(pageURL string, baseURL *neturl.URL, bodyBytes []byte, browserPreviewHTML string) (*CollectedWebPageResult, error) {
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

	contentNode := findWeChatContentNode(body)

	contentHTML := strings.TrimSpace(renderWeChatChildren(contentNode, baseURL, imageResolver, 0))
	contentText := collapseCollectorWhitespace(extractWeChatText(contentNode))
	if contentHTML == "" {
		contentHTML = strings.TrimSpace(renderWeChatChildren(body, baseURL, imageResolver, 0))
	}
	if contentText == "" {
		contentText = collapseCollectorWhitespace(extractWeChatText(body))
	}
	if contentHTML == "" && contentText == "" {
		return nil, fmt.Errorf("页面正文为空，暂时无法生成草稿")
	}

	title := firstNonEmpty(meta.title, extractWeChatHeadingText(contentNode), extractWeChatHeadingText(body))

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

func findWeChatContentNode(body *xhtml.Node) *xhtml.Node {
	if node := findWeChatNodeByID(body, "js_content"); node != nil {
		return node
	}

	if node := findWeChatNodeByClass(body, "rich_media_content"); node != nil {
		return node
	}

	if node := findWeChatNodeByClass(body, "rich_media_area_primary"); node != nil {
		return node
	}

	if node := findWeChatNodeByClass(body, "rich_media_area_primary_inner"); node != nil {
		return node
	}

	return body
}

func findWeChatNodeByID(root *xhtml.Node, id string) *xhtml.Node {
	var result *xhtml.Node
	walkCollectorNode(root, func(node *xhtml.Node, _ int) bool {
		if node.Type == xhtml.ElementNode && strings.EqualFold(attrValue(node, "id"), id) {
			result = node
			return false
		}
		return true
	})
	return result
}

func findWeChatNodeByClass(root *xhtml.Node, class string) *xhtml.Node {
	var result *xhtml.Node
	walkCollectorNode(root, func(node *xhtml.Node, _ int) bool {
		if node.Type == xhtml.ElementNode {
			for _, c := range strings.Fields(attrValue(node, "class")) {
				if strings.EqualFold(c, class) {
					result = node
					return false
				}
			}
		}
		return true
	})
	return result
}

func renderWeChatChildren(node *xhtml.Node, baseURL *neturl.URL, imageResolver *collectorImageResolver, depth int) string {
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(renderWeChatNode(child, baseURL, imageResolver, depth))
	}
	return builder.String()
}

func renderWeChatNode(node *xhtml.Node, baseURL *neturl.URL, imageResolver *collectorImageResolver, depth int) string {
	switch node.Type {
	case xhtml.TextNode:
		text := collapseCollectorWhitespace(node.Data)
		if text == "" {
			return ""
		}
		return stdhtml.EscapeString(text)
	case xhtml.ElementNode:
		if isWeChatExcludedNode(node) || depth > 64 {
			return ""
		}

		tag := node.Data
		if !isCollectorAllowedTag(tag) {
			return renderWeChatChildren(node, baseURL, imageResolver, depth+1)
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

		content := renderWeChatChildren(node, baseURL, imageResolver, depth+1)
		content = strings.TrimSpace(content)
		if content == "" && (tag == "p" || tag == "li" || tag == "blockquote" || strings.HasPrefix(tag, "h")) {
			return ""
		}

		return "<" + tag + attrs.String() + ">" + content + "</" + tag + ">"
	default:
		return ""
	}
}

func isWeChatExcludedNode(node *xhtml.Node) bool {
	if node.Type != xhtml.ElementNode {
		return false
	}

	if collectorNodeLooksHidden(node) {
		return true
	}

	switch node.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Iframe, atom.Svg:
		return true
	}

	return false
}

func extractWeChatText(node *xhtml.Node) string {
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
			if isWeChatExcludedNode(current) {
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

func extractWeChatHeadingText(node *xhtml.Node) string {
	var result string
	walkCollectorNode(node, func(current *xhtml.Node, _ int) bool {
		if current.Type != xhtml.ElementNode {
			return true
		}
		switch current.DataAtom {
		case atom.H1, atom.H2:
			text := collapseCollectorWhitespace(extractWeChatText(current))
			if text != "" {
				result = text
				return false
			}
		}
		return true
	})
	return result
}
