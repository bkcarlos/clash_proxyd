package core

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/clash-proxyd/proxyd/internal/logx"

	"go.uber.org/zap"
)

var semverPattern = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)

// Stage constants for progress reporting.
const (
	StageFetchRelease = "fetch_release" // querying GitHub release API
	StageDownload     = "download"      // downloading binary archive
	StageExtract      = "extract"       // extracting / preparing binary
	StageInstall      = "install"       // atomic swap on disk
	StageRestart      = "restart"       // restarting mihomo process
	StageDone         = "done"
	StageError        = "error"
)

// ProgressEvent is emitted at each installation phase.
type ProgressEvent struct {
	Stage   string // one of the Stage* constants
	Percent int    // 0-100; only meaningful during StageDownload
	Message string
}

// ProgressFunc receives progress events. It must not block.
type ProgressFunc func(ProgressEvent)

// UpdaterConfig represents updater settings.
type UpdaterConfig struct {
	Enabled       bool
	CheckOnStart  bool
	ReleaseAPI    string
	DownloadDir   string
	TargetVersion string
}

// UpdateResult describes updater execution result.
type UpdateResult struct {
	Updated        bool
	OldVersion     string
	NewVersion     string
	BinaryPath     string
	BackupPath     string
	DownloadedFrom string
}

type releaseResponse struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Updater checks and installs mihomo updates.
type Updater struct {
	cfg        UpdaterConfig
	httpClient *http.Client
}

// NewUpdater creates updater.
func NewUpdater(cfg UpdaterConfig) *Updater {
	return &Updater{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

// DetectBinaryVersion returns the version string reported by the mihomo binary,
// or an empty string with an error if the binary is missing or unexecutable.
func DetectBinaryVersion(binaryPath string) (string, error) {
	return detectBinaryVersion(binaryPath)
}

// FetchLatestVersion queries the release API and returns the latest version tag.
func (u *Updater) FetchLatestVersion(ctx context.Context) (string, error) {
	release, err := u.fetchRelease(ctx)
	if err != nil {
		return "", err
	}
	v := sanitizeVersion(release.TagName)
	if v == "" {
		return "", fmt.Errorf("empty version tag from release api")
	}
	return v, nil
}

// FetchVersionList queries the releases list API and returns up to limit version tags.
// Prerelease entries are included; pass limit ≤ 0 to default to 20.
func (u *Updater) FetchVersionList(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}

	listURL, err := u.resolveListURL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create releases list request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query releases list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("releases list api returned status %d: %s", resp.StatusCode, string(body))
	}

	var releases []releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases list: %w", err)
	}

	versions := make([]string, 0, limit)
	for _, r := range releases {
		v := sanitizeVersion(r.TagName)
		if v == "" {
			continue
		}
		versions = append(versions, v)
		if len(versions) >= limit {
			break
		}
	}
	return versions, nil
}

// resolveListURL derives the paginated releases list URL from the configured release API.
func (u *Updater) resolveListURL() (string, error) {
	parsed, err := url.Parse(u.cfg.ReleaseAPI)
	if err != nil {
		return "", fmt.Errorf("invalid release api: %w", err)
	}
	// Strip /latest or /tags/... to get the base /releases path.
	if idx := strings.Index(parsed.Path, "/releases"); idx >= 0 {
		parsed.Path = parsed.Path[:idx+len("/releases")]
	}
	parsed.RawQuery = "per_page=30"
	return parsed.String(), nil
}

// CheckAndUpdateWithProgress is like CheckAndUpdate but reports progress.
func (u *Updater) CheckAndUpdateWithProgress(ctx context.Context, binaryPath string, progress ProgressFunc) (UpdateResult, error) {
	emit := func(stage, msg string, pct int) {
		if progress != nil {
			progress(ProgressEvent{Stage: stage, Percent: pct, Message: msg})
		}
	}

	result := UpdateResult{BinaryPath: binaryPath}
	if strings.TrimSpace(binaryPath) == "" {
		return result, fmt.Errorf("binary path is required")
	}

	emit(StageFetchRelease, "Fetching release information…", 0)

	currentVersion, err := detectBinaryVersion(binaryPath)
	if err != nil {
		logx.Warn("Failed to detect current mihomo version", zap.String("binary", binaryPath), zap.Error(err))
	}
	result.OldVersion = currentVersion

	release, err := u.fetchRelease(ctx)
	if err != nil {
		return result, err
	}

	latestVersion := sanitizeVersion(release.TagName)
	if latestVersion == "" {
		return result, fmt.Errorf("empty release tag from api")
	}
	result.NewVersion = latestVersion

	if !u.NeedsUpdate(currentVersion, latestVersion) {
		emit(StageDone, fmt.Sprintf("Already up to date (%s)", currentVersion), 100)
		return result, nil
	}

	asset, err := selectAsset(release.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return result, err
	}
	result.DownloadedFrom = asset.BrowserDownloadURL

	emit(StageDownload, "Starting download…", 0)
	backupPath, err := u.downloadAndInstallWithProgress(ctx, asset, binaryPath, progress)
	if err != nil {
		return result, err
	}

	result.BackupPath = backupPath
	result.Updated = true

	logx.Info("Mihomo binary updated",
		zap.String("old_version", result.OldVersion),
		zap.String("new_version", result.NewVersion),
		zap.String("binary", result.BinaryPath))

	return result, nil
}

// CheckAndUpdate checks latest release and installs update when needed.
func (u *Updater) CheckAndUpdate(ctx context.Context, binaryPath string) (UpdateResult, error) {
	result := UpdateResult{BinaryPath: binaryPath}

	if strings.TrimSpace(binaryPath) == "" {
		return result, fmt.Errorf("binary path is required")
	}
	if strings.TrimSpace(u.cfg.ReleaseAPI) == "" {
		return result, fmt.Errorf("release api is required")
	}
	if strings.TrimSpace(u.cfg.DownloadDir) == "" {
		return result, fmt.Errorf("download dir is required")
	}

	currentVersion, err := detectBinaryVersion(binaryPath)
	if err != nil {
		logx.Warn("Failed to detect current mihomo version", zap.String("binary", binaryPath), zap.Error(err))
	}
	result.OldVersion = currentVersion

	release, err := u.fetchRelease(ctx)
	if err != nil {
		return result, err
	}

	latestVersion := sanitizeVersion(release.TagName)
	if latestVersion == "" {
		return result, fmt.Errorf("empty release tag from api")
	}
	result.NewVersion = latestVersion

	if !u.NeedsUpdate(currentVersion, latestVersion) {
		logx.Info("Mihomo binary is already up-to-date",
			zap.String("current_version", currentVersion),
			zap.String("latest_version", latestVersion))
		return result, nil
	}

	asset, err := selectAsset(release.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return result, err
	}

	result.DownloadedFrom = asset.BrowserDownloadURL
	backupPath, err := u.DownloadAndInstall(ctx, asset, binaryPath)
	if err != nil {
		return result, err
	}

	result.BackupPath = backupPath
	result.Updated = true

	logx.Info("Mihomo binary updated",
		zap.String("old_version", result.OldVersion),
		zap.String("new_version", result.NewVersion),
		zap.String("binary", result.BinaryPath),
		zap.String("asset", asset.Name))

	return result, nil
}

// Install downloads and installs the specified version (or latest when version
// is empty), regardless of the currently installed version.
func (u *Updater) Install(ctx context.Context, binaryPath, version string) (UpdateResult, error) {
	result := UpdateResult{BinaryPath: binaryPath}

	if strings.TrimSpace(binaryPath) == "" {
		return result, fmt.Errorf("binary path is required")
	}

	// Temporarily override target version if provided.
	origTarget := u.cfg.TargetVersion
	if version != "" {
		u.cfg.TargetVersion = version
	}
	defer func() { u.cfg.TargetVersion = origTarget }()

	currentVersion, _ := detectBinaryVersion(binaryPath)
	result.OldVersion = currentVersion

	release, err := u.fetchRelease(ctx)
	if err != nil {
		return result, err
	}

	latestVersion := sanitizeVersion(release.TagName)
	if latestVersion == "" {
		return result, fmt.Errorf("empty release tag from api")
	}
	result.NewVersion = latestVersion

	asset, err := selectAsset(release.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return result, err
	}

	result.DownloadedFrom = asset.BrowserDownloadURL
	backupPath, err := u.DownloadAndInstall(ctx, asset, binaryPath)
	if err != nil {
		return result, err
	}

	result.BackupPath = backupPath
	result.Updated = true

	logx.Info("Mihomo binary installed",
		zap.String("version", result.NewVersion),
		zap.String("binary", result.BinaryPath))

	return result, nil
}

// NeedsUpdate compares versions.
func (u *Updater) NeedsUpdate(currentVersion, latestVersion string) bool {
	current := parseSemver(currentVersion)
	latest := parseSemver(latestVersion)

	if latest == nil {
		return false
	}
	if current == nil {
		return true
	}
	for i := 0; i < 3; i++ {
		if latest[i] > current[i] {
			return true
		}
		if latest[i] < current[i] {
			return false
		}
	}
	return false
}

// InstallWithProgress downloads and installs the given version (or latest when
// version is empty) and reports progress through the provided callback.
func (u *Updater) InstallWithProgress(ctx context.Context, binaryPath, version string, progress ProgressFunc) (UpdateResult, error) {
	emit := func(stage, msg string, pct int) {
		if progress != nil {
			progress(ProgressEvent{Stage: stage, Percent: pct, Message: msg})
		}
	}

	result := UpdateResult{BinaryPath: binaryPath}

	origTarget := u.cfg.TargetVersion
	if version != "" {
		u.cfg.TargetVersion = version
	}
	defer func() { u.cfg.TargetVersion = origTarget }()

	currentVersion, _ := detectBinaryVersion(binaryPath)
	result.OldVersion = currentVersion

	emit(StageFetchRelease, "Fetching release information…", 0)
	release, err := u.fetchRelease(ctx)
	if err != nil {
		return result, err
	}

	latestVersion := sanitizeVersion(release.TagName)
	if latestVersion == "" {
		return result, fmt.Errorf("empty release tag from api")
	}
	result.NewVersion = latestVersion

	asset, err := selectAsset(release.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return result, err
	}
	result.DownloadedFrom = asset.BrowserDownloadURL

	emit(StageDownload, "Starting download…", 0)
	backupPath, err := u.downloadAndInstallWithProgress(ctx, asset, binaryPath, progress)
	if err != nil {
		return result, err
	}

	result.BackupPath = backupPath
	result.Updated = true

	logx.Info("Mihomo binary installed",
		zap.String("version", result.NewVersion),
		zap.String("binary", result.BinaryPath))

	emit(StageDone, fmt.Sprintf("Installed %s", result.NewVersion), 100)
	return result, nil
}

// DownloadAndInstall downloads and atomically installs release asset.
func (u *Updater) DownloadAndInstall(ctx context.Context, asset releaseAsset, targetPath string) (string, error) {
	return u.downloadAndInstallWithProgress(ctx, asset, targetPath, nil)
}

func (u *Updater) downloadAndInstallWithProgress(ctx context.Context, asset releaseAsset, targetPath string, progress ProgressFunc) (string, error) {
	emit := func(stage, msg string, pct int) {
		if progress != nil {
			progress(ProgressEvent{Stage: stage, Percent: pct, Message: msg})
		}
	}

	if err := os.MkdirAll(u.cfg.DownloadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download dir: %w", err)
	}

	downloadPath := filepath.Join(u.cfg.DownloadDir, asset.Name)
	if err := u.downloadFile(ctx, asset.BrowserDownloadURL, downloadPath, progress); err != nil {
		return "", err
	}

	emit(StageExtract, "Extracting binary…", 100)

	installTemp := targetPath + ".new"
	if err := prepareInstalledBinary(downloadPath, installTemp, asset.Name); err != nil {
		return "", err
	}

	emit(StageInstall, "Installing binary…", 100)

	backupPath := targetPath + ".bak"
	_ = os.Remove(backupPath)
	if _, statErr := os.Stat(targetPath); statErr == nil {
		// Binary exists — back it up before replacing.
		if err := os.Rename(targetPath, backupPath); err != nil {
			_ = os.Remove(installTemp)
			return "", fmt.Errorf("failed to backup current binary: %w", err)
		}
	} else {
		// Fresh install — no binary to back up.
		backupPath = ""
	}

	if err := os.Rename(installTemp, targetPath); err != nil {
		_ = os.Rename(backupPath, targetPath)
		return "", fmt.Errorf("failed to replace binary: %w", err)
	}

	return backupPath, nil
}

// Rollback restores backup binary. If BackupPath is empty (fresh install had no
// prior binary) the installed binary is simply removed.
func (u *Updater) Rollback(result UpdateResult) error {
	if result.BinaryPath == "" {
		return fmt.Errorf("rollback failed: binary path is unknown")
	}
	if result.BackupPath == "" {
		// No backup means this was a fresh install — just remove what we placed.
		_ = os.Remove(result.BinaryPath)
		return nil
	}

	if _, err := os.Stat(result.BackupPath); err != nil {
		return fmt.Errorf("backup binary not found: %w", err)
	}

	_ = os.Remove(result.BinaryPath)
	if err := os.Rename(result.BackupPath, result.BinaryPath); err != nil {
		return fmt.Errorf("failed to restore backup binary: %w", err)
	}

	logx.Warn("Rolled back mihomo binary",
		zap.String("binary", result.BinaryPath),
		zap.String("backup", result.BackupPath),
		zap.String("version", result.OldVersion))

	return nil
}

func (u *Updater) fetchRelease(ctx context.Context) (*releaseResponse, error) {
	releaseURL, err := u.resolveReleaseURL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query release api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("release api returned status %d: %s", resp.StatusCode, string(body))
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release response: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("release api returned empty tag")
	}

	return &release, nil
}

func (u *Updater) resolveReleaseURL() (string, error) {
	if strings.TrimSpace(u.cfg.TargetVersion) == "" {
		return u.cfg.ReleaseAPI, nil
	}

	parsed, err := url.Parse(u.cfg.ReleaseAPI)
	if err != nil {
		return "", fmt.Errorf("invalid release api: %w", err)
	}

	if strings.HasSuffix(parsed.Path, "/latest") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/latest") + "/tags/" + url.PathEscape(u.cfg.TargetVersion)
		return parsed.String(), nil
	}

	return u.cfg.ReleaseAPI, nil
}

func (u *Updater) downloadFile(ctx context.Context, sourceURL, targetPath string, progress ProgressFunc) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download release asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asset download failed with status %d: %s", resp.StatusCode, string(body))
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create download file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = resp.Body
	total := resp.ContentLength // -1 when unknown
	if progress != nil && total > 0 {
		reader = &progressReader{
			Reader: resp.Body,
			total:  total,
			onRead: func(read, tot int64) {
				pct := int(read * 100 / tot)
				progress(ProgressEvent{Stage: StageDownload, Percent: pct,
					Message: fmt.Sprintf("Downloading: %d%%", pct)})
			},
		}
	}

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write download file: %w", err)
	}

	return nil
}

type progressReader struct {
	io.Reader
	total  int64
	read   int64
	onRead func(read, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)
	if pr.onRead != nil {
		pr.onRead(pr.read, pr.total)
	}
	return n, err
}

func prepareInstalledBinary(downloadPath, targetPath, assetName string) error {
	lowerName := strings.ToLower(assetName)

	switch {
	case strings.HasSuffix(lowerName, ".tar.gz") || strings.HasSuffix(lowerName, ".tgz"):
		if err := extractBinaryFromTarGz(downloadPath, targetPath); err != nil {
			return err
		}
	case strings.HasSuffix(lowerName, ".gz"):
		if err := extractBinaryFromGzip(downloadPath, targetPath); err != nil {
			return err
		}
	default:
		if err := copyFile(downloadPath, targetPath); err != nil {
			return err
		}
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod installed binary: %w", err)
	}

	return nil
}

func extractBinaryFromGzip(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open gzip asset: %w", err)
	}
	defer source.Close()

	gzReader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("failed to open gzip stream: %w", err)
	}
	defer gzReader.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target binary: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, gzReader); err != nil {
		return fmt.Errorf("failed to extract gzip asset: %w", err)
	}

	return nil
}

func extractBinaryFromTarGz(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open archive asset: %w", err)
	}
	defer source.Close()

	gzReader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("failed to open archive gzip stream: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := strings.ToLower(filepath.Base(header.Name))
		if strings.Contains(name, "mihomo") || name == "clash" || name == "mihomo" {
			target, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create target binary: %w", err)
			}
			defer target.Close()

			if _, err := io.Copy(target, tarReader); err != nil {
				return fmt.Errorf("failed to extract binary from archive: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("archive does not contain mihomo binary")
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

func selectAsset(assets []releaseAsset, goos, goarch string) (releaseAsset, error) {
	if len(assets) == 0 {
		return releaseAsset{}, fmt.Errorf("release has no downloadable assets")
	}

	osToken := strings.ToLower(goos)
	archToken := normalizeArch(goarch)

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "sha256") || strings.Contains(name, "checksum") {
			continue
		}
		if strings.Contains(name, osToken) && strings.Contains(name, archToken) {
			return asset, nil
		}
	}

	return releaseAsset{}, fmt.Errorf("no compatible asset found for %s/%s", goos, goarch)
}

func normalizeArch(arch string) string {
	switch arch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "386":
		return "386"
	default:
		return strings.ToLower(arch)
	}
}

func detectBinaryVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "-v")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute binary version command: %w", err)
	}

	version := sanitizeVersion(string(out))
	if version == "" {
		return "", fmt.Errorf("failed to parse version from output: %s", strings.TrimSpace(string(out)))
	}

	return version, nil
}

func sanitizeVersion(raw string) string {
	match := semverPattern.FindStringSubmatch(raw)
	if len(match) != 4 {
		return ""
	}
	return fmt.Sprintf("%s.%s.%s", match[1], match[2], match[3])
}

func parseSemver(raw string) []int {
	match := semverPattern.FindStringSubmatch(raw)
	if len(match) != 4 {
		return nil
	}
	result := make([]int, 0, 3)
	for i := 1; i <= 3; i++ {
		n, err := strconv.Atoi(match[i])
		if err != nil {
			return nil
		}
		result = append(result, n)
	}
	return result
}
