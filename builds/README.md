# YouTube Summarizer

An enterprise-grade YouTube video summarizer built in Go that fetches videos from specified channels, generates AI-powered summaries using Claude, and sends email digests.

## 🚀 Features

- **Multi-Channel Monitoring**: Track videos from multiple YouTube channels
- **AI-Powered Summaries**: Generate concise summaries using Claude API
- **Excel Data Storage**: Store channel data, processed videos, and summaries in Excel format
- **Email Digests**: Send beautiful HTML email digests (daily/weekly)
- **Cross-Platform**: Single executable for Windows, macOS, and Linux
- **Enterprise Architecture**: Clean, maintainable Go code with proper interfaces
- **Concurrent Processing**: Efficient parallel processing of multiple videos
- **Graceful Error Handling**: Robust error handling and logging

## 📋 Prerequisites

- Go 1.19+ installed
- YouTube Data API v3 key
- Claude API key (Anthropic)
- Optional: RapidAPI key for transcript fetching
- Optional: Email credentials for digest functionality

## 🛠 Installation & Setup

### 1. Clone and Build

```bash
git clone <repository-url>
cd youtube-summarizer
go build -o youtube-summarizer ./cmd/summarizer
```

### 2. Environment Configuration

Create a `.env` file in the root directory:

```bash
# Required API Keys
YOUTUBE_API_KEY=your_youtube_api_key_here
CLAUDE_API_KEY=your_claude_api_key_here

# Optional: Transcript fetching
RAPID_API_KEY=your_rapidapi_key_here

# Optional: Email functionality
EMAIL_USERNAME=your.email@gmail.com
EMAIL_PASSWORD=your_app_password_here
```

### 3. Configure Channels

The application will create a `youtube-data.xlsx` file on first run. Add your YouTube channels to the "Channels" sheet:

| ID | Name | Username |
|---|---|---|
| UCxxxxxx | Channel Name | @channelhandle |

You can find channel IDs from YouTube URLs or using the YouTube API.

## 🏃‍♂️ Usage

### Basic Usage

```bash
# Run with default settings
./youtube-summarizer

# Run in development mode
./youtube-summarizer -dev

# Test email configuration
./youtube-summarizer -test-email

# Use custom configuration
./youtube-summarizer -config ./custom-config.yaml -excel ./my-data.xlsx
```

### Command Line Options

```
-config string    Path to configuration file (default: "configs/config.yaml")
-env string       Path to environment file (default: ".env")
-excel string     Path to Excel data file (default: "youtube-data.xlsx")
-test-email       Send test email and exit
-dev              Run in development mode with verbose logging
-help             Show help message
```

## ⚙️ Configuration

The application uses `configs/config.yaml` for configuration:

```yaml
app:
  check_frequency: "daily"  # daily, weekly, hourly
  email_frequency: "weekly" # daily, weekly

youtube:
  max_videos_per_channel: 5

processing:
  max_concurrent_videos: 3
  transcript_timeout: "30s"

email:
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  subject_template: "YouTube Summary - {date}"

ai:
  max_transcript_length: 15000
  summary_prompt: |
    Video Title: "{title}". Summarize the key takeaways from the following video 
    transcript into a concise paragraph. Focus on the main points and actionable advice:
    
    {transcript}
```

## 🏗 Architecture

```
youtube-summarizer/
├── cmd/summarizer/         # Main application entry point
├── internal/
│   ├── clients/           # API clients (YouTube, Claude, Transcript)
│   ├── config/            # Configuration management
│   ├── logger/            # Structured logging
│   ├── services/          # Business logic (Processor, Email)
│   └── storage/           # Data persistence (Excel)
├── pkg/types/             # Shared interfaces and types
└── configs/               # Configuration files
```

## 🔧 Cross-Platform Builds

Build for multiple platforms:

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o youtube-summarizer-windows.exe ./cmd/summarizer

# macOS
GOOS=darwin GOARCH=amd64 go build -o youtube-summarizer-mac ./cmd/summarizer

# Linux
GOOS=linux GOARCH=amd64 go build -o youtube-summarizer-linux ./cmd/summarizer
```

## 📊 Excel File Structure

The application uses Excel files with three sheets:

1. **Channels**: YouTube channels to monitor
2. **ProcessedVideos**: Tracks processed video IDs
3. **Summaries**: Stores video summaries with status

## 📧 Email Digests

Email digests are sent in beautiful HTML format containing:

- Video titles and channel names
- AI-generated summaries
- Direct links to videos
- Summary statistics

## 🔐 API Keys Setup

### YouTube Data API v3
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable YouTube Data API v3
4. Create credentials (API Key)

### Claude API
1. Sign up at [Anthropic](https://console.anthropic.com/)
2. Generate an API key
3. Add to environment variables

### RapidAPI (Optional)
1. Sign up at [RapidAPI](https://rapidapi.com/)
2. Subscribe to YouTube Transcriptor API
3. Get your API key

## 🚀 Production Deployment

For production use:

1. Use a process manager (systemd, supervisor)
2. Set up log rotation
3. Configure firewall rules if needed
4. Use environment-specific configuration files
5. Set up monitoring and alerting

## 📝 Logging

The application uses structured logging with different levels:

- **Debug**: Detailed operation information
- **Info**: General operation status
- **Warn**: Warning conditions
- **Error**: Error conditions

Logs are output to stdout/stderr and can be redirected to files.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Troubleshooting

### Common Issues

1. **API Rate Limits**: Implement proper rate limiting and retries
2. **Transcript Unavailable**: Falls back to mock transcripts if RapidAPI fails
3. **Email Delivery**: Check SMTP settings and app passwords for Gmail
4. **Excel File Permissions**: Ensure the application has write access to the Excel file

### Support

For issues and questions:

1. Check the logs for detailed error information
2. Verify API keys and configuration
3. Test with `-test-email` flag for email issues
4. Run with `-dev` flag for verbose logging

---

Built with ❤️ using Go, Claude API, and enterprise best practices.