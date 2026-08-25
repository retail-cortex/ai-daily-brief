package mailer

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"ai-daily-brief/internal/database"
)

var (
	boldRegex   = regexp.MustCompile(`\*\*(.*?)\*\*`)
	italicRegex = regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`)
	codeRegex   = regexp.MustCompile("`([^`]+)`")
	linkRegex   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func formatInlineMarkdown(text string) string {
	escaped := html.EscapeString(text)

	// Bold: **text**
	escaped = boldRegex.ReplaceAllString(escaped, `<strong style="color: #ffffff; font-weight: 700;">$1</strong>`)

	// Code: `code`
	escaped = codeRegex.ReplaceAllString(escaped, `<code style="background-color: #0f172a; border: 1px solid #334155; color: #a5b4fc; padding: 2px 5px; border-radius: 4px; font-size: 12px; font-family: monospace;">$1</code>`)

	// Links: [text](url)
	escaped = linkRegex.ReplaceAllString(escaped, `<a href="$2" target="_blank" style="color: #38bdf8; text-decoration: underline;">$1</a>`)

	// Italic: *text*
	escaped = italicRegex.ReplaceAllString(escaped, `<em style="color: #cbd5e1; font-style: italic;">$1$2</em>`)

	return escaped
}

// MarkdownToInlineHTML converts markdown text into semantic HTML with inline styles suitable for email clients
func MarkdownToInlineHTML(md string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	inList := false
	listType := "" // "ul" or "ol"

	closeListIfNeeded := func() {
		if inList {
			if listType == "ul" {
				sb.WriteString("</ul>\n")
			} else {
				sb.WriteString("</ol>\n")
			}
			inList = false
			listType = ""
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			closeListIfNeeded()
			continue
		}

		// Horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			closeListIfNeeded()
			sb.WriteString(`<hr style="border: 0; height: 1px; background-color: #334155; margin: 16px 0;" />` + "\n")
			continue
		}

		// Headers
		if strings.HasPrefix(trimmed, "### ") {
			closeListIfNeeded()
			hText := strings.TrimPrefix(trimmed, "### ")
			sb.WriteString(`<h3 style="color: #c4b5fd; font-size: 15px; font-weight: 700; margin: 16px 0 6px 0; border-bottom: 1px solid #334155; padding-bottom: 4px;">` + formatInlineMarkdown(hText) + `</h3>` + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			closeListIfNeeded()
			hText := strings.TrimPrefix(trimmed, "## ")
			sb.WriteString(`<h2 style="color: #93c5fd; font-size: 17px; font-weight: 700; margin: 18px 0 8px 0; border-bottom: 1px solid #334155; padding-bottom: 4px;">` + formatInlineMarkdown(hText) + `</h2>` + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			closeListIfNeeded()
			hText := strings.TrimPrefix(trimmed, "# ")
			sb.WriteString(`<h1 style="color: #a5b4fc; font-size: 19px; font-weight: 800; margin: 20px 0 10px 0;">` + formatInlineMarkdown(hText) + `</h1>` + "\n")
			continue
		}

		// Blockquotes
		if strings.HasPrefix(trimmed, "> ") {
			closeListIfNeeded()
			qText := strings.TrimPrefix(trimmed, "> ")
			sb.WriteString(`<blockquote style="border-left: 3px solid #6366f1; background-color: rgba(30, 27, 75, 0.4); padding: 8px 12px; margin: 8px 0; border-radius: 0 8px 8px 0; color: #c7d2fe; font-size: 13px;">` + formatInlineMarkdown(qText) + `</blockquote>` + "\n")
			continue
		}

		// Unordered list items: * or -
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			if !inList || listType != "ul" {
				closeListIfNeeded()
				sb.WriteString(`<ul style="margin: 8px 0; padding-left: 20px; list-style-type: disc;">` + "\n")
				inList = true
				listType = "ul"
			}
			liText := trimmed[2:]
			sb.WriteString(`<li style="margin-bottom: 6px; color: #e2e8f0; font-size: 13px; line-height: 1.6;">` + formatInlineMarkdown(liText) + `</li>` + "\n")
			continue
		}

		// Regular paragraph
		closeListIfNeeded()
		sb.WriteString(`<p style="margin: 6px 0; color: #cbd5e1; font-size: 13px; line-height: 1.6;">` + formatInlineMarkdown(trimmed) + `</p>` + "\n")
	}

	closeListIfNeeded()
	return sb.String()
}

func renderCard(item database.NewsItem, badgeColor string) string {
	var sb strings.Builder
	sb.WriteString(`<div style="background-color: #1e293b; border-radius: 12px; padding: 18px; margin-bottom: 14px; border: 1px solid #334155;">`)
	sb.WriteString(`<div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px;">`)
	sb.WriteString(`<span style="background-color: ` + badgeColor + `; color: #ffffff; padding: 3px 8px; border-radius: 6px; font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px;">`)
	sb.WriteString(html.EscapeString(item.Company))
	sb.WriteString(`</span>`)
	sb.WriteString(`<span style="color: #94a3b8; font-size: 11px; font-weight: 500;">Pub: ` + html.EscapeString(item.PubDate) + `</span>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`<h3 style="margin: 0 0 8px 0; color: #f8fafc; font-size: 15px; line-height: 1.4; font-weight: 600;">`)
	sb.WriteString(html.EscapeString(item.Title))
	sb.WriteString(`</h3>`)
	sb.WriteString(`<p style="margin: 0 0 12px 0; color: #cbd5e1; font-size: 13px; line-height: 1.5;">`)
	sb.WriteString(html.EscapeString(item.Summary))
	sb.WriteString(`</p>`)
	sb.WriteString(`<div>`)
	sb.WriteString(`<a href="` + html.EscapeString(item.Link) + `" target="_blank" style="display: inline-block; background-color: #0284c7; color: #ffffff; padding: 6px 14px; border-radius: 6px; font-size: 12px; font-weight: 600; text-decoration: none;">`)
	sb.WriteString(`Read Full Source &rarr;</a>`)
	sb.WriteString(`</div></div>`)
	return sb.String()
}

func GenerateNewsletterHTML(items []database.NewsItem, dateStr string) string {
	var frontier, gcp, papers, business, oss []database.NewsItem

	for _, item := range items {
		switch item.Category {
		case database.CategoryFrontierModels:
			frontier = append(frontier, item)
		case database.CategoryGoogleCloud:
			gcp = append(gcp, item)
		case database.CategoryResearchPapers:
			papers = append(papers, item)
		case database.CategoryBusinessInfra:
			business = append(business, item)
		case database.CategoryOSSTooling:
			oss = append(oss, item)
		}
	}

	renderSection := func(title, color string, list []database.NewsItem, badgeColor string) string {
		if len(list) == 0 {
			return ""
		}
		var sb strings.Builder
		sb.WriteString(`<div style="margin-bottom: 28px;">`)
		sb.WriteString(`<h2 style="color: ` + color + `; font-size: 17px; border-bottom: 2px solid #334155; padding-bottom: 6px; margin-bottom: 14px;">`)
		sb.WriteString(title)
		sb.WriteString(`</h2>`)
		for _, item := range list {
			sb.WriteString(renderCard(item, badgeColor))
		}
		sb.WriteString(`</div>`)
		return sb.String()
	}

	// Fetch cached LLM TLDR from settings if available
	var tldrSetting database.Setting
	tldrText := ""
	if database.DB != nil {
		if err := database.DB.First(&tldrSetting, "key = ?", "latest_tldr").Error; err == nil && tldrSetting.Value != "" {
			tldrText = tldrSetting.Value
		}
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Daily AI & Cloud Intelligence Digest</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #090d16; color: #f8fafc; margin: 0; padding: 20px;">
  <div style="max-width: 680px; margin: 0 auto; background-color: #090d16;">
    <!-- Header Banner -->
    <div style="background: linear-gradient(135deg, #1e1b4b 0%, #311b92 50%, #0f172a 100%); border-radius: 16px; padding: 28px 20px; text-align: center; border: 1px solid #4338ca; margin-bottom: 24px;">
      <div style="font-size: 28px; font-weight: 800; color: #ffffff; letter-spacing: -0.5px; margin-bottom: 6px;">
        ⚡ Daily AI & Cloud Intelligence Digest
      </div>
      <div style="color: #c7d2fe; font-size: 13px; font-weight: 500;">
        Frontier Models &bull; Google Cloud &bull; Research Papers &bull; Business &bull; ` + html.EscapeString(dateStr) + `
      </div>
      <div style="display: flex; justify-content: center; flex-wrap: wrap; gap: 12px; margin-top: 14px; color: #a5b4fc; font-size: 12px;">
        <span>🔵 ` + fmt.Sprintf("%d", len(frontier)) + ` Models</span>
        <span>&bull;</span>
        <span>☁️ ` + fmt.Sprintf("%d", len(gcp)) + ` Cloud</span>
        <span>&bull;</span>
        <span>🟣 ` + fmt.Sprintf("%d", len(papers)) + ` Papers</span>
        <span>&bull;</span>
        <span>🟢 ` + fmt.Sprintf("%d", len(business)) + ` Business</span>
        <span>&bull;</span>
        <span>🟠 ` + fmt.Sprintf("%d", len(oss)) + ` Tooling</span>
      </div>
    </div>`)

	// LLM Executive TL;DR Card formatted as inline-styled HTML
	if tldrText != "" {
		formattedTLDR := MarkdownToInlineHTML(tldrText)
		sb.WriteString(`
    <div style="background: linear-gradient(135deg, #0f172a 0%, #1e1b4b 100%); border: 1px solid #6366f1; border-radius: 14px; padding: 20px; margin-bottom: 28px;">
      <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 12px;">
        <span style="font-size: 18px;">🤖</span>
        <h2 style="margin: 0; color: #a5b4fc; font-size: 16px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px;">
          Gemini 3.7 Executive TL;DR & Strategic Analysis
        </h2>
      </div>
      <div style="color: #e2e8f0; font-size: 13px; line-height: 1.6;">` + formattedTLDR + `</div>
    </div>`)
	}

	// 1. Frontier Models
	sb.WriteString(renderSection("🔵 Frontier Models (Google, Anthropic, OpenAI, X AI, Meta)", "#60a5fa", frontier, "#2563eb"))

	// 2. Google Cloud Release Notes & Vertex AI
	sb.WriteString(renderSection("☁️ Google Cloud & Vertex AI Release Notes (docs.cloud.google.com)", "#38bdf8", gcp, "#0284c7"))

	// 3. AI Research Papers
	sb.WriteString(renderSection("🟣 AI Research Papers & Breakthroughs (arXiv & Hugging Face)", "#c084fc", papers, "#9333ea"))

	// 4. AI Business & Infrastructure
	sb.WriteString(renderSection("🟢 AI Business, Funding & Compute Infrastructure", "#34d399", business, "#059669"))

	// 5. OSS & Tooling
	sb.WriteString(renderSection("🟠 Open-Source Models & Developer Tooling", "#fbbf24", oss, "#d97706"))

	// Footer
	sb.WriteString(`
    <div style="text-align: center; color: #64748b; font-size: 11px; border-top: 1px solid #334155; padding-top: 16px; margin-top: 28px;">
      <p style="margin: 0 0 4px 0;">Automated AI & Cloud Intelligence Go Agent</p>
      <p style="margin: 0;">Generated on ` + html.EscapeString(dateStr) + ` &bull; High-Performance Native Go Runtime</p>
    </div>
  </div>
</body>
</html>`)

	return sb.String()
}
