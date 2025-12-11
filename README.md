<div align="center">

# 🔐 Clipleaks

**Real-time clipboard monitoring for sensitive information**

[![Gitleaks](https://github.com/ahokinson/clipleaks/actions/workflows/gitleaks.yml/badge.svg)](https://github.com/ahokinson/clipleaks/actions/workflows/gitleaks.yml)
[![CodeQL](https://github.com/ahokinson/clipleaks/actions/workflows/codeql.yml/badge.svg)](https://github.com/ahokinson/clipleaks/actions/workflows/codeql.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/ahokinson/clipleaks)](https://go.dev/)
[![License](https://img.shields.io/github/license/ahokinson/clipleaks)](https://github.com/ahokinson/clipleaks/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/ahokinson/clipleaks)](https://github.com/ahokinson/clipleaks/releases)

Protect yourself from accidentally pasting secrets. Clipleaks monitors your clipboard in real-time and alerts you when it detects API keys, tokens, private keys, and other secrets.

Inspired by [gitleaks](https://github.com/gitleaks/gitleaks) 🔍

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [Configuration](#configuration)

</div>

---

## ✨ Features

- ⏱️ **Real-time Monitoring** - Continuously watches your clipboard for sensitive data
- 🧠 **Smart Detection** - Combines pattern matching with entropy-based analysis
- 🔔 **Native Notifications** - Get instant desktop alerts when secrets are detected
- 🎯 **Comprehensive Coverage** - Detects 11+ types of credentials including:
  - AWS Access Keys
  - GitHub Tokens (PAT & Classic)
  - Slack Tokens & Webhooks
  - Stripe API Keys
  - Google API Keys
  - Private Keys (RSA, DSA, EC)
  - JWT Tokens
  - Generic API Keys
  - Passwords in URLs
- 🍎 **Cross-Platform** - Works on macOS and Linux
- 🪶 **Lightweight** - Single binary with no external dependencies
- ⚙️ **Configurable** - Customize detection thresholds, polling intervals, and disable specific patterns
- 🔑 **Privacy-First** - All processing happens locally; nothing leaves your machine

## 📦 Installation

### Download Pre-built Binary

Download the latest release for your platform from the [releases page](https://github.com/ahokinson/clipleaks/releases).

Extract the binary and move it to `/usr/local/bin/` to make it available system-wide.

### Prerequisites

**macOS:** Works out of the box.

**Linux:** Install clipboard and notification tools.

```bash
# Ubuntu/Debian
sudo apt-get install xclip libnotify-bin

# Fedora/RHEL
sudo dnf install xclip libnotify

# Arch Linux
sudo pacman -S xclip libnotify
```

## 🚀 Usage

### Quick Start

Run Clipleaks in the foreground to see detections in real-time:

```bash
clipleaks
```

Now copy any text containing secrets to your clipboard and watch for alerts!

### Command Line Options

```bash
clipleaks [options]

Options:
  -i    Install Clipleaks as a background service
  -s    Check service status
  -u    Uninstall Clipleaks background service
  -v    Show version information
```

### Running as a Background Service

Install Clipleaks to run automatically in the background:

```bash
# Install and start the service
clipleaks -i

# Check if it's running
clipleaks -s

# Uninstall the service
clipleaks -u
```

**Service Details:**

- **macOS:** Installed as a LaunchAgent (`~/Library/LaunchAgents/com.clipleaks.plist`)
- **Linux:** Installed as a systemd user service (`~/.config/systemd/user/clipleaks.service`)

## ⚙️ Configuration

Clipleaks creates a configuration file at `~/.config/clipleaks/config.json` on first run. You can customize its behavior by editing this file.

```json
{
  "entropy_threshold": 3.5,
  "enable_entropy": true,
  "poll_interval_ms": 500,
  "max_clipboard_size": 1048576,
  "pattern_match_size": 10240,
  "pattern_overlap_size": 200,
  "disabled_patterns": []
}
```

### Configuration Options

| Option                 | Type  | Default   | Range       |
| ---------------------- | ----- | --------- | ----------- |
| `entropy_threshold`    | float | `3.5`     | 0.0 - 8.0   |
| `enable_entropy`       | bool  | `true`    | -           |
| `poll_interval_ms`     | int   | `500`     | 100 - 60000 |
| `max_clipboard_size`   | int   | `1048576` | 1KB - 10MB  |
| `pattern_match_size`   | int   | `10240`   | 1KB - 1MB   |
| `pattern_overlap_size` | int   | `200`     | 0 - 1KB     |
| `disabled_patterns`    | array | `[]`      | -           |

### Detected Patterns

You can disable specific patterns by adding their IDs to the `disabled_patterns` array:

| Pattern ID           | Description                                |
| -------------------- | ------------------------------------------ |
| `aws-access-key`     | AWS Access Key IDs                         |
| `github-token`       | GitHub Personal Access Tokens (new format) |
| `github-pat-classic` | GitHub Personal Access Tokens (classic)    |
| `generic-api-key`    | Generic API keys                           |
| `slack-token`        | Slack API tokens                           |
| `slack-webhook`      | Slack webhook URLs                         |
| `stripe-api-key`     | Stripe API keys                            |
| `google-api-key`     | Google API keys                            |
| `private-key`        | Private key headers (RSA, DSA, EC)         |
| `password-in-url`    | Passwords embedded in URLs                 |
| `jwt-token`          | JSON Web Tokens                            |

## 🤝 Contributing

Contributions are welcome! Here are some ways you can help:

- Report bugs and issues
- Suggest new features or detection patterns
- Improve documentation
- Write tests

All contributions should be submitted via pull requests.

## ⚠️ Disclaimer

Clipleaks is a security tool designed to help prevent accidental exposure of sensitive information. While it provides an additional layer of protection, it should not be relied upon as the sole security measure. Always follow security best practices when handling credentials and sensitive data.
