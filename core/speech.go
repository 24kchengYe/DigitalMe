package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SpeechToText transcribes audio to text.
type SpeechToText interface {
	Transcribe(ctx context.Context, audio []byte, format string, lang string) (string, error)
}

// SpeechConfig holds STT configuration for the engine.
type SpeechCfg struct {
	Enabled  bool
	Provider string
	Language string
	STT      SpeechToText
}

// OpenAIWhisper implements SpeechToText using the OpenAI-compatible Whisper API.
// Works with OpenAI, Groq, and any endpoint that implements the same multipart API.
type OpenAIWhisper struct {
	APIKey  string
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewOpenAIWhisper(apiKey, baseURL, model string) *OpenAIWhisper {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "whisper-1"
	}
	return &OpenAIWhisper{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (w *OpenAIWhisper) Transcribe(ctx context.Context, audio []byte, format string, lang string) (string, error) {
	ext := formatToExt(format)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "audio."+ext)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("write audio: %w", err)
	}
	_ = writer.WriteField("model", w.Model)
	_ = writer.WriteField("response_format", "text")
	if lang != "" {
		_ = writer.WriteField("language", lang)
	}
	writer.Close()

	url := w.BaseURL + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+w.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := w.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper API %d: %s", resp.StatusCode, string(body))
	}

	// response_format=text returns plain text; try to handle JSON fallback
	text := strings.TrimSpace(string(body))
	if strings.HasPrefix(text, "{") {
		var jr struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(body, &jr) == nil {
			text = jr.Text
		}
	}
	return text, nil
}

// LocalWhisper implements SpeechToText using a local whisper.cpp executable.
// No server needed — calls the CLI directly.
type LocalWhisper struct {
	ExePath   string // path to whisper-cli executable
	ModelPath string // path to ggml model file
}

func NewLocalWhisper(exePath, modelPath string) *LocalWhisper {
	return &LocalWhisper{ExePath: exePath, ModelPath: modelPath}
}

func (w *LocalWhisper) Transcribe(ctx context.Context, audio []byte, format string, lang string) (string, error) {
	tmpDir := os.TempDir()

	// Write audio to a temp file
	audioFile := filepath.Join(tmpDir, "digitalme_stt_input."+formatToExt(format))
	if err := os.WriteFile(audioFile, audio, 0o644); err != nil {
		return "", fmt.Errorf("local whisper: write temp audio: %w", err)
	}
	defer os.Remove(audioFile)

	// Convert to WAV 16kHz mono (whisper.cpp requirement)
	wavFile := filepath.Join(tmpDir, "digitalme_stt_input.wav")
	ffmpegPath, err := FindFFmpeg()
	if err != nil {
		return "", fmt.Errorf("local whisper: ffmpeg required for audio conversion: %w", err)
	}
	convCmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", audioFile,
		"-ar", "16000",
		"-ac", "1",
		"-sample_fmt", "s16",
		"-f", "wav",
		"-y", wavFile,
	)
	var convErr bytes.Buffer
	convCmd.Stderr = &convErr
	if err := convCmd.Run(); err != nil {
		return "", fmt.Errorf("local whisper: ffmpeg convert: %w (%s)", err, convErr.String())
	}
	defer os.Remove(wavFile)

	// Call whisper-cli
	args := []string{
		"-m", w.ModelPath,
		"-f", wavFile,
		"--no-timestamps",
		"-nt", // no timestamps in text output
	}
	if lang != "" {
		args = append(args, "-l", lang)
	}

	cmd := exec.CommandContext(ctx, w.ExePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("local whisper: running", "exe", w.ExePath, "model", w.ModelPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("local whisper: %w (stderr: %s)", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		// whisper.cpp sometimes outputs to stderr channel
		text = strings.TrimSpace(stderr.String())
	}

	// Clean up whisper.cpp output artifacts (sometimes outputs "[BLANK_AUDIO]" etc.)
	if text == "[BLANK_AUDIO]" {
		return "", nil
	}

	return text, nil
}

// ConvertAudioToMP3 uses ffmpeg to convert audio from unsupported formats to mp3.
// Returns the mp3 bytes. If ffmpeg is not installed, returns an error.
func ConvertAudioToMP3(audio []byte, srcFormat string) ([]byte, error) {
	ffmpegPath, err := FindFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: install ffmpeg or set ffmpeg_path in [speech.local] config")
	}

	cmd := exec.Command(ffmpegPath,
		"-i", "pipe:0",
		"-f", srcFormat,
		"-f", "mp3",
		"-ac", "1",
		"-ar", "16000",
		"-y",
		"pipe:1",
	)
	// For formats where ffmpeg can't auto-detect from pipe, specify input format
	if srcFormat == "amr" || srcFormat == "silk" {
		cmd = exec.Command(ffmpegPath,
			"-f", srcFormat,
			"-i", "pipe:0",
			"-f", "mp3",
			"-ac", "1",
			"-ar", "16000",
			"-y",
			"pipe:1",
		)
	} else {
		cmd = exec.Command(ffmpegPath,
			"-i", "pipe:0",
			"-f", "mp3",
			"-ac", "1",
			"-ar", "16000",
			"-y",
			"pipe:1",
		)
	}

	cmd.Stdin = bytes.NewReader(audio)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// NeedsConversion returns true if the audio format is not directly supported by Whisper API.
func NeedsConversion(format string) bool {
	switch strings.ToLower(format) {
	case "mp3", "mp4", "mpeg", "mpga", "m4a", "wav", "webm":
		return false
	default:
		return true
	}
}

// ffmpegOverride stores an explicit ffmpeg path from config. Set at startup.
var ffmpegOverride string

// SetFFmpegPath sets an explicit path to the ffmpeg executable (from config).
func SetFFmpegPath(path string) {
	ffmpegOverride = path
}

// FindFFmpeg locates the ffmpeg executable. It checks:
// 1. Explicit config path (ffmpeg_path in config.toml)
// 2. PATH (exec.LookPath)
// 3. Common Windows install locations
func FindFFmpeg() (string, error) {
	// 1. Explicit config path
	if ffmpegOverride != "" {
		if _, err := os.Stat(ffmpegOverride); err == nil {
			return ffmpegOverride, nil
		}
	}

	// 2. Standard PATH lookup
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}

	// 3. Common Windows locations
	for _, candidate := range []string{
		`D:\ffmpeg\ffmpeg.exe`,
		`D:\ffmpeg\bin\ffmpeg.exe`,
		`C:\ffmpeg\ffmpeg.exe`,
		`C:\ffmpeg\bin\ffmpeg.exe`,
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("ffmpeg not found: install ffmpeg or set ffmpeg_path in [speech.local] config")
}

// HasFFmpeg checks if ffmpeg is available.
func HasFFmpeg() bool {
	_, err := FindFFmpeg()
	return err == nil
}

func formatToExt(format string) string {
	switch strings.ToLower(format) {
	case "amr":
		return "amr"
	case "ogg", "oga", "opus":
		return "ogg"
	case "m4a", "mp4", "aac":
		return "m4a"
	case "mp3":
		return "mp3"
	case "wav":
		return "wav"
	case "webm":
		return "webm"
	case "silk":
		return "silk"
	default:
		return format
	}
}

// TranscribeAudio is a convenience function used by the Engine.
// It handles format conversion (if needed) and calls the STT provider.
func TranscribeAudio(ctx context.Context, stt SpeechToText, audio *AudioAttachment, lang string) (string, error) {
	data := audio.Data
	format := strings.ToLower(audio.Format)

	if NeedsConversion(format) {
		slog.Debug("speech: converting audio", "from", format, "to", "mp3")
		converted, err := ConvertAudioToMP3(data, format)
		if err != nil {
			return "", err
		}
		data = converted
		format = "mp3"
	}

	slog.Debug("speech: transcribing", "format", format, "size", len(data))
	return stt.Transcribe(ctx, data, format, lang)
}
