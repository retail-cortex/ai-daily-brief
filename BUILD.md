# 🛠️ Build & Compilation Guide

This document details how to compile, test, and build the **AI Daily Brief** from source using the modern, hermetic **Bazel 9** build system.

---

## 📋 Prerequisites

Because the project uses a **hermetic build system** (Bazel Bzlmod), you do **not** need to install Go, Node.js, or pnpm on your local machine to build the project. The build system automatically downloads the exact required toolchain versions:
*   **Go SDK**: `1.26.3`
*   **Node.js**: `22.12.0`
*   **pnpm**: `10.24.0`

The **only** prerequisite is **Bazel** (or the wrapper **Bazelisk**):
- **macOS** (Homebrew): `brew install bazelisk`
- **Linux / Windows**: Install `bazelisk` via your system package manager or download it directly from [GitHub](https://github.com/bazelbuild/bazelisk).

---

## 🏗️ 1. Building with Bazel (Recommended)

### Build the entire project
This compiles the React frontend assets, embeds them into the Go binary package, and builds the executables for your host system:
```bash
bazel build //...
```

### Run the application locally
Compile and launch the daily brief agent directly under the Bazel environment:
```bash
bazel run //:run
```
*(Alternatively, you can run the full target: `bazel run //cmd/ai-daily-brief`)*


### Run all tests
Run all backend Go unit tests and frontend Vitest component tests in isolated, hermetic sandboxes:
```bash
bazel test //...
```

### Collect test coverage
Calculate test coverage for code instrumentation in the Go backend packages:
```bash
bazel coverage //internal/...
```

---

## 🚀 2. Cross-Platform Compilation

Bazel compiles single-file static executables for all target platforms simultaneously from any host operating system:

```bash
# Build for Apple Silicon Mac (ARM64)
bazel build //cmd/ai-daily-brief:ai_daily_brief_darwin_arm64

# Build for Intel Mac (x86_64)
bazel build //cmd/ai-daily-brief:ai_daily_brief_darwin_amd64

# Build for Linux (x86_64)
bazel build //cmd/ai-daily-brief:ai_daily_brief_linux_amd64

# Build for Windows (x86_64)
bazel build //cmd/ai-daily-brief:ai_daily_brief_windows_amd64
```
The compiled binaries will be output to `bazel-bin/cmd/ai-daily-brief/` in folders named `<target_name>_/`.

### Standalone Distribution Archives
To bundle the compiled executable side-by-side with the default configuration `.env.toml` file (which is required to run the application), build the packaged archive targets:

```bash
# Package for Apple Silicon Mac (.tar.gz)
bazel build //cmd/ai-daily-brief:pkg_darwin_arm64

# Package for Intel Mac (.tar.gz)
bazel build //cmd/ai-daily-brief:pkg_darwin_amd64

# Package for Linux (.tar.gz)
bazel build //cmd/ai-daily-brief:pkg_linux_amd64
# Package for Windows (.zip)
bazel build //cmd/ai-daily-brief:pkg_windows_amd64

# Package for Linux / Chromebook (.deb Installer)
bazel build //cmd/ai-daily-brief:pkg_deb
```
The output archives and installer packages (e.g., `ai-daily-brief-darwin-arm64.tar.gz`, `ai-daily-brief_1.0.0_all.deb`) will be generated at `bazel-bin/cmd/ai-daily-brief/`.

---

## 💻 3. Standard Go & npm Compilation (Manual)

If you prefer to compile natively without using Bazel, you will need Go `1.26+`, Node.js `22+`, and `pnpm` installed on your host system:

```bash
# 1. Install frontend dependencies
cd web
pnpm install

# 2. Compile React frontend assets to dist/
pnpm build
cd ..

# 3. Compile the Go binary (attaches web/dist/ automatically via embed)
go build -o bin/ai-daily-brief ./cmd/ai-daily-brief
```
