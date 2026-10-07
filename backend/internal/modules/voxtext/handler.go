package voxtext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"backend_voxtext/internal/modules/middleware"
)

const multipartMemoryLimit = 32 << 20

type ServiceInterface interface {
	Transcribe(ctx context.Context, inputPath string) ([]DiarizedUtterance, error)
}

type Handler struct {
	service        ServiceInterface
	maxUploadBytes int64
	uploadDir      string
}

type transcriptionResponse struct {
	Utterances []DiarizedUtterance `json:"utterances"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(service ServiceInterface, maxUploadBytes int64, uploadDir string) *Handler {
	return &Handler{
		service:        service,
		maxUploadBytes: maxUploadBytes,
		uploadDir:      uploadDir,
	}
}

func (h *Handler) Transcribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method must be POST")
		return
	}
	if h == nil || h.service == nil || h.maxUploadBytes <= 0 || strings.TrimSpace(h.uploadDir) == "" {
		writeError(w, http.StatusServiceUnavailable, "transcription service is not ready")
		return
	}
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := r.ParseMultipartForm(multipartMemoryLimit); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "uploaded file is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "request must be multipart/form-data")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "multipart field \"file\" is required")
		return
	}
	defer file.Close()

	if err := os.MkdirAll(h.uploadDir, 0o700); err != nil {
		log.Printf("create upload directory: %v", err)
		writeError(w, http.StatusInternalServerError, "could not store uploaded file")
		return
	}

	temporaryFile, err := os.CreateTemp(h.uploadDir, "stt-upload-*"+safeExtension(header.Filename))
	if err != nil {
		log.Printf("create upload file: %v", err)
		writeError(w, http.StatusInternalServerError, "could not store uploaded file")
		return
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)

	copyErr := copyAndClose(temporaryFile, file)
	if copyErr != nil {
		log.Printf("store uploaded file: %v", copyErr)
		writeError(w, http.StatusInternalServerError, "could not store uploaded file")
		return
	}

	utterances, err := h.service.Transcribe(r.Context(), temporaryPath)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeError(w, http.StatusRequestTimeout, "transcription was canceled")
			return
		}
		if errors.Is(err, ErrServiceClosed) {
			writeError(w, http.StatusServiceUnavailable, "transcription service is shutting down")
			return
		}
		log.Printf("transcribe %q: %v", filepath.Base(header.Filename), err)
		writeError(w, http.StatusInternalServerError, "could not transcribe audio")
		return
	}
	if utterances == nil {
		utterances = make([]DiarizedUtterance, 0)
	}

	writeJSON(w, http.StatusOK, transcriptionResponse{Utterances: utterances})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func copyAndClose(destination *os.File, source io.Reader) error {
	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		return fmt.Errorf("copy upload: %w", err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("close upload: %w", err)
	}
	return nil
}

func safeExtension(filename string) string {
	extension := strings.ToLower(filepath.Ext(filepath.Base(filename)))
	if len(extension) > 16 {
		return ""
	}
	for _, character := range strings.TrimPrefix(extension, ".") {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			return ""
		}
	}
	return extension
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}
