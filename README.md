# ⚡ AI Daily Brief

A high-performance, single-binary AI news and research paper intelligence agent powered by **Go, Gin, GORM, SQLite, and embedded React**. It aggregates frontier model announcements, arXiv research papers, AI venture capital deals, GPU datacenter infrastructure, and open-source tooling into a unified tabular dashboard and daily HTML executive newsletter.

---

- **Google Gemini 3.7 & Vertex AI Agent Engine**:
  - **Model Selection**: Defaults to **`gemini-3.7-flash`**, with support for `gemini-3.1-pro`, `gemini-3.5-flash`, `gemini-3.5-flash-lite`, `gemini-2.5-flash`, `gemini-2.5-pro`, and custom models.
  - **Dual Auth**: Google AI Studio API Key OR Google Cloud Vertex AI with Application Default Credentials (ADC).
  - **Interactive Chat Assistant**: Context-grounded conversational agent with on-demand full webpage extraction to answer questions about any article or release note.
  - **Automated Daily TL;DR**: LLM-generated executive analysis across all 5 streams embedded in the dashboard and exported briefs.
- **5 Intelligence Streams**:
  - 🔵 **Frontier Models**: Google (Gemini 3.7 / 2.0), Anthropic (Claude), OpenAI (GPT-5 / o3 / Sora), X AI (Grok), Meta AI.
  - ☁️ **Google Cloud**: Official release notes, Vertex AI infrastructure, AI Hypercomputer, TPUs, and GKE.
  - 🟣 **AI Research Papers**: Multi-category arXiv API (`cs.CL`, `cs.AI`, `cs.CV`, `cs.LG`), Hugging Face Daily Papers, and academic benchmarks.
  - 🟢 **AI Business & Infra**: Funding rounds, datacenter buildouts (Nvidia Blackwell, Colossus), hyperscaler cloud deals.
  - 🟠 **OSS & Tooling**: Open weights (DeepSeek, Llama, Qwen, Mistral), local inference runtimes (vLLM, Ollama), fine-tuning harnesses.
- **Sub-Second Parallel Crawling**: Concurrent goroutines crawl all sources in parallel in **< 1 second** (`~900ms`).
- **Strict SHA-256 Deduplication**: Content URL hashing guarantees non-repeated items.
- **Zero-Dependency Single Executable**: React frontend is embedded directly in memory with `//go:embed`.
- **Daily HTML Newsletter**: 4-part categorized email briefing and simulation sandbox.
- **Cross-Platform**: Native binaries for macOS (Apple Silicon & Intel), Linux (x86_64 & ARM64), and Windows (`.exe`).

---

## 🚀 Quick Start (Running via Bazel)

The recommended way to compile and run the application locally is using **Bazel** (or the wrapper **Bazelisk**), which provides a fully hermetic build of the React frontend and Go backend:

```bash
# Clone the repository
git clone <repo_url>
cd ai-daily-brief

# Run the hermetic Go server and auto-open the web dashboard
bazel run //:run
```

When started, the application launches the Gin server and automatically opens **`http://localhost:3001`** in your default web browser.

---

## 📦 Native Desktop Installers (Pre-Compiled Releases)

If you prefer to install the application as a native desktop utility rather than building it from source, download the pre-compiled installer for your operating system from the **GitHub Releases** page:

*   **macOS (Apple Silicon M1/M2/M3/M4)**: Download `ai-daily-brief-darwin-arm64.dmg`. Double-click to mount the disk image and drag the app into your `/Applications` directory.
*   **Windows (64-bit)**: Download `ai-daily-brief-setup.exe`. Double-click to run the setup wizard, which will install the server, set up default configuration settings, and register Start Menu shortcuts.
*   **Linux / Chromebook (Debian/Ubuntu)**: Download `ai-daily-brief_1.0.0_all.deb`. Double-click the file (directly from the Files app on ChromeOS) to install the program and register a shortcut launcher in your desktop/Crostini App Drawer.

For complete compilation and cross-platform build commands, see [BUILD.md](file:///Users/ryan/Projects/retail-cortex/ai-daily-brief/BUILD.md).

---

## ⚙️ CLI Options & Flags

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-port <number>` | Custom HTTP port to bind | `3001` |
| `-no-browser` | Disable automatic browser launch on startup | `false` |
| `-batch` | Run crawler once headlessly, print summary, and exit | `false` |

### Examples

*   **Run on a custom port without auto-opening the browser:**
    ```bash
    bazel run //:run -- -port 8080 -no-browser
    ```

*   **Run as a headless CLI batch cron job (ideal for crontab/CI):**
    ```bash
    bazel run //:run -- -batch
    ```

---

## 📄 Executive Newsletter Export

The **Daily Executive Digest** tab allows you to preview, copy, or download the full categorized intelligence briefing:
- **Interactive Responsive Preview**: Switch between Desktop and Mobile preview modes.
- **Copy HTML**: Copy the raw HTML template with inline CSS directly to your clipboard for pasting into any email client or report.
- **Download `.html`**: Export the digest as a standalone `ai_cloud_intelligence_digest_YYYY-MM-DD.html` file.

---

## 📁 Directory Structure

```text
├── cmd/
│   └── ai-daily-brief/
│       └── main.go       # Main entry point & CLI handler
├── internal/
│   ├── agent/            # Gemini client & content enricher
│   ├── config/           # Environment & TOML config loader
│   ├── crawler/          # Parallel goroutine news crawlers
│   ├── database/         # SQLite GORM schema & db connection
│   ├── mailer/           # HTML email digest builder
│   └── server/           # Gin REST API & static web router
├── web/                  # React 19 + TypeScript + Tailwind frontend
├── patches/              # Bzlmod patches for external dependencies
├── BUILD.bazel           # Root Bazel build definitions
├── MODULE.bazel          # Bazel 9 Bzlmod dependency configuration
└── BUILD.md              # Hermetic Bazel compilation guide
```
