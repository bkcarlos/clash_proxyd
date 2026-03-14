package api

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// DownloadLog streams the full log file as an attachment.
// Query param: source = "proxyd" | "mihomo"
func (h *Handler) DownloadLog(c *gin.Context) {
	source := c.DefaultQuery("source", "proxyd")

	var logFile string
	switch source {
	case "proxyd":
		logFile = h.proxydLogFile
	case "mihomo":
		logFile = h.resolvedMihomoLogFile()
	default:
		h.respondError(c, http.StatusBadRequest, "Invalid source: must be proxyd or mihomo")
		return
	}

	if logFile == "" {
		h.respondError(c, http.StatusNotFound, "Log file not configured")
		return
	}

	if _, err := os.Stat(logFile); err != nil {
		h.respondError(c, http.StatusNotFound, "Log file not found: "+logFile)
		return
	}

	filename := fmt.Sprintf("%s.log", source)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.File(logFile)
}

// GetLogs returns the last N lines from proxyd or mihomo log file.
// Query params:
//   - source: "proxyd" | "mihomo"  (default: proxyd)
//   - lines:  number of tail lines (default: 200, max: 2000)
func (h *Handler) GetLogs(c *gin.Context) {
	source := c.DefaultQuery("source", "proxyd")
	linesStr := c.DefaultQuery("lines", "200")

	lines, err := strconv.Atoi(linesStr)
	if err != nil || lines <= 0 {
		lines = 200
	}
	if lines > 2000 {
		lines = 2000
	}

	var logFile string
	switch source {
	case "proxyd":
		logFile = h.proxydLogFile
	case "mihomo":
		logFile = h.resolvedMihomoLogFile()
	default:
		h.respondError(c, http.StatusBadRequest, "Invalid source: must be proxyd or mihomo")
		return
	}

	if logFile == "" {
		h.respondJSON(c, http.StatusOK, gin.H{
			"source":    source,
			"file":      "",
			"lines":     []string{},
			"available": false,
			"message":   "Log file not configured",
		})
		return
	}

	tail, fileSize, err := tailFile(logFile, lines)
	if err != nil {
		if os.IsNotExist(err) {
			h.respondJSON(c, http.StatusOK, gin.H{
				"source":    source,
				"file":      logFile,
				"lines":     []string{},
				"available": false,
				"message":   fmt.Sprintf("Log file not found: %s", logFile),
			})
			return
		}
		h.respondError(c, http.StatusInternalServerError, "Failed to read log file: "+err.Error())
		return
	}

	h.respondJSON(c, http.StatusOK, gin.H{
		"source":    source,
		"file":      logFile,
		"lines":     tail,
		"total":     len(tail),
		"file_size": fileSize,
		"available": true,
	})
}

// resolvedMihomoLogFile finds the mihomo log file in the configured log directory.
func (h *Handler) resolvedMihomoLogFile() string {
	if h.mihomoLogDir == "" {
		return ""
	}

	// Try known file names first.
	candidates := []string{"mihomo.log", "clash.log", "core.log"}
	for _, name := range candidates {
		p := filepath.Join(h.mihomoLogDir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to first .log file in the directory that isn't proxyd.log.
	entries, err := os.ReadDir(h.mihomoLogDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".log") && name != filepath.Base(h.proxydLogFile) {
			return filepath.Join(h.mihomoLogDir, name)
		}
	}
	return ""
}

// tailFile reads the last n lines from path and returns them along with file size.
func tailFile(path string, n int) ([]string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	fileSize := info.Size()

	// For files small enough to read entirely, scan line by line.
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	ring := make([]string, n)
	idx := 0
	total := 0

	for scanner.Scan() {
		ring[idx%n] = scanner.Text()
		idx++
		total++
	}
	if err := scanner.Err(); err != nil {
		return nil, fileSize, err
	}

	if total == 0 {
		return []string{}, fileSize, nil
	}

	// Reconstruct in order.
	result := make([]string, 0, min(total, n))
	if total <= n {
		result = append(result, ring[:total]...)
	} else {
		start := idx % n
		for i := 0; i < n; i++ {
			result = append(result, ring[(start+i)%n])
		}
	}
	return result, fileSize, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
