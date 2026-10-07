package voxtext

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"backend_voxtext/internal/modules/middleware"
)

type testVerifier struct{}

func (testVerifier) VerifyAccessToken(raw string) (int64, error) {
	if raw != "valid-token" {
		return 0, errors.New("invalid token")
	}
	return 42, nil
}

func serveAuthenticated(handler *Handler, response *httptest.ResponseRecorder, request *http.Request) {
	request.Header.Set("Authorization", "Bearer valid-token")
	middleware.RequireAuth(testVerifier{})(http.HandlerFunc(handler.Transcribe)).ServeHTTP(response, request)
}

type fakeService struct {
	result       []DiarizedUtterance
	err          error
	receivedPath string
	receivedData []byte
}

func (s *fakeService) Transcribe(_ context.Context, inputPath string) ([]DiarizedUtterance, error) {
	s.receivedPath = inputPath
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	s.receivedData = data
	return s.result, s.err
}

func TestTranscribeReturnsJSONAndRemovesUpload(t *testing.T) {
	uploadDir := t.TempDir()
	service := &fakeService{result: []DiarizedUtterance{{
		Start: 0.1, End: 1.2, Speaker: 2, Text: "hello",
	}}}
	handler := NewHandler(service, 1<<20, uploadDir)
	request := multipartRequest(t, "/v1/transcriptions", "../../outside.mp3", []byte("audio"))
	response := httptest.NewRecorder()

	serveAuthenticated(handler, response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d, body %s", response.Code, response.Body.String())
	}
	if string(service.receivedData) != "audio" {
		t.Fatalf("uploaded data: got %q", service.receivedData)
	}
	if filepath.Dir(service.receivedPath) != uploadDir {
		t.Fatalf("upload directory: got %q, want %q", filepath.Dir(service.receivedPath), uploadDir)
	}
	if _, err := os.Stat(service.receivedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary upload was not removed: %v", err)
	}

	var body transcriptionResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Utterances) != 1 || body.Utterances[0].Text != "hello" {
		t.Fatalf("unexpected response: %+v", body)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("content type: got %q", contentType)
	}
}

func TestTranscribeRejectsOversizedRequest(t *testing.T) {
	handler := NewHandler(&fakeService{}, 32, t.TempDir())
	request := multipartRequest(t, "/v1/transcriptions", "wav", bytes.Repeat([]byte("x"), 128))
	response := httptest.NewRecorder()

	serveAuthenticated(handler, response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status: got %d, want %d; body %s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
}

func TestTranscribeHandlesServiceError(t *testing.T) {
	handler := NewHandler(&fakeService{err: errors.New("inference failed")}, 1<<20, t.TempDir())
	request := multipartRequest(t, "/v1/transcriptions", "wav", []byte("audio"))
	response := httptest.NewRecorder()

	serveAuthenticated(handler, response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d; body %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func TestTranscribeRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeService{}, 1<<20, t.TempDir())
	request := multipartRequest(t, "/v1/transcriptions", "audio.wav", []byte("audio"))
	response := httptest.NewRecorder()
	handler.Transcribe(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func multipartRequest(t *testing.T, target, filename string, data []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, target, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
