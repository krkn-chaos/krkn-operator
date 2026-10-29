package api

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	pathpkg "path"
	"regexp"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
)

const maxKrknAIManifestSize = 16 << 20

var krknAISHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type krknAIArchiveManifest struct {
	Files []krknAIArchiveManifestFile `json:"files"`
}

type krknAIArchiveManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// @Summary Download Krkn-AI run results archive
// @Description Download every file in the committed artifact manifest as one ZIP after target authorization.
// @Tags krkn-ai
// @Produce application/zip
// @Param name path string true "KrknAIRun name"
// @Success 200 {file} file "Committed run artifacts ZIP"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run or committed artifacts not found"
// @Failure 502 {object} ErrorResponse "Committed artifact manifest is corrupt"
// @Failure 503 {object} ErrorResponse "Artifact upload is updating or service unavailable"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name}/results/download [get]
func (h *Handler) downloadKrknAIRunArchive(w http.ResponseWriter, r *http.Request, name string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}

	basePath := "/" + krknaiserver.EscapePath("v1/runs/"+string(run.UID))
	manifestResponse, err := h.artifactClient.Do(r.Context(), http.MethodGet, basePath+"/results", nil)
	if err != nil {
		writeKrknAIServiceUnavailable(w)
		return
	}
	if manifestResponse.StatusCode != http.StatusOK {
		copyKrknAIArchiveServiceError(w, manifestResponse)
		return
	}
	manifestBytes, readErr := io.ReadAll(io.LimitReader(manifestResponse.Body, maxKrknAIManifestSize+1))
	closeErr := manifestResponse.Body.Close()
	if readErr != nil || closeErr != nil {
		writeKrknAIServiceUnavailable(w)
		return
	}
	if len(manifestBytes) > maxKrknAIManifestSize {
		writeKrknAIArchiveCorrupt(w, "Krkn-AI returned an oversized artifact manifest")
		return
	}
	var manifest krknAIArchiveManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		writeKrknAIArchiveCorrupt(w, "Krkn-AI returned an invalid artifact manifest")
		return
	}
	if manifest.Files == nil {
		writeKrknAIArchiveCorrupt(w, "Krkn-AI returned an invalid artifact manifest")
		return
	}

	seen := make(map[string]struct{}, len(manifest.Files))
	for i := range manifest.Files {
		file := &manifest.Files[i]
		if !safeKrknAIArchivePath(file.Path) {
			writeKrknAIArchiveCorrupt(w, "Krkn-AI manifest contains an unsafe artifact path")
			return
		}
		if _, duplicate := seen[file.Path]; duplicate {
			writeKrknAIArchiveCorrupt(w, "Krkn-AI manifest contains duplicate artifact paths")
			return
		}
		seen[file.Path] = struct{}{}
		if file.Size < 0 || file.Size == int64(1<<63-1) || !krknAISHA256Pattern.MatchString(file.SHA256) {
			writeKrknAIArchiveCorrupt(w, "Krkn-AI manifest contains invalid artifact metadata")
			return
		}
	}

	archiveFile, err := os.CreateTemp("", "krkn-ai-results-*.zip")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to create Krkn-AI results archive"})
		return
	}
	archivePath := archiveFile.Name()
	defer os.Remove(archivePath)
	defer archiveFile.Close()
	zipWriter := zip.NewWriter(archiveFile)
	for _, file := range manifest.Files {
		if r.Context().Err() != nil {
			_ = zipWriter.Close()
			return
		}
		if err := h.appendKrknAIArchiveFile(r, basePath, zipWriter, file); err != nil {
			_ = zipWriter.Close()
			if r.Context().Err() != nil {
				return
			}
			if err == errKrknAIArchiveSnapshotChanged {
				writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "artifact_updating", Message: "Krkn-AI artifact snapshot changed during download"})
				return
			}
			if serviceErr, ok := err.(*krknAIArchiveServiceError); ok {
				copyKrknAIArchiveServiceError(w, serviceErr.response)
				return
			}
			if _, ok := err.(*krknAIArchiveUpstreamError); ok {
				writeKrknAIServiceUnavailable(w)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to create Krkn-AI results archive"})
			return
		}
	}
	if err := zipWriter.Close(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to finalize Krkn-AI results archive"})
		return
	}
	info, err := archiveFile.Stat()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to finalize Krkn-AI results archive"})
		return
	}
	if r.Context().Err() != nil {
		return
	}

	if _, err := archiveFile.Seek(0, io.SeekStart); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to read Krkn-AI results archive"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": run.Name + "-results.zip"}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, archiveFile)
}

var errKrknAIArchiveSnapshotChanged = fmt.Errorf("Krkn-AI artifact snapshot changed")

type krknAIArchiveServiceError struct {
	response *http.Response
}

func (*krknAIArchiveServiceError) Error() string {
	return "Krkn-AI artifact service returned an error"
}

type krknAIArchiveUpstreamError struct{}

func (*krknAIArchiveUpstreamError) Error() string {
	return "failed to read Krkn-AI artifact"
}

func (h *Handler) appendKrknAIArchiveFile(r *http.Request, basePath string, zipWriter *zip.Writer, file krknAIArchiveManifestFile) error {
	response, err := h.artifactClient.Do(r.Context(), http.MethodGet, basePath+"/files/"+krknaiserver.EscapePath(file.Path), nil)
	if err != nil {
		return &krknAIArchiveUpstreamError{}
	}
	if response.StatusCode != http.StatusOK {
		return &krknAIArchiveServiceError{response: response}
	}
	defer response.Body.Close()

	entry, err := zipWriter.Create(file.Path)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(entry, hash), io.LimitReader(response.Body, file.Size+1))
	if err != nil {
		return &krknAIArchiveUpstreamError{}
	}
	if written != file.Size || hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
		return errKrknAIArchiveSnapshotChanged
	}
	return nil
}

func safeKrknAIArchivePath(name string) bool {
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00") || pathpkg.IsAbs(name) || pathpkg.Clean(name) != name {
		return false
	}
	first := name
	if slash := strings.IndexByte(name, '/'); slash >= 0 {
		first = name[:slash]
	}
	if len(first) >= 2 && first[1] == ':' && ((first[0] >= 'a' && first[0] <= 'z') || (first[0] >= 'A' && first[0] <= 'Z')) {
		return false
	}
	return true
}

func copyKrknAIArchiveServiceError(w http.ResponseWriter, response *http.Response) {
	defer response.Body.Close()
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func writeKrknAIArchiveCorrupt(w http.ResponseWriter, message string) {
	writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "artifact_error", Message: message})
}
