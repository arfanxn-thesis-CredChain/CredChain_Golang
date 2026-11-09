package pyai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"CredChain_Golang/config"
)

// ErrVerifyFileUnprocessable means the Python AI service accepted the /verify
// request (HTTP 200) but could not verify the file itself — it rejected the
// upload or extracted no readable text. This is a document problem the user can
// fix, NOT the AI service being down (transport/decode failures return other
// errors). Callers should surface a "we couldn't read your document" message.
var ErrVerifyFileUnprocessable = errors.New("verify: file could not be processed")

// ExtractedID is one identifier extracted from a credential document by Python.
type ExtractedID struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ExtractFile is a single file to send to Python /extract, /verify, or /extract-ids.
// If MIMEType is empty, the client derives it from Filename.
type ExtractFile struct {
	Filename string
	MIMEType string
	Data     []byte
}

// PythonExtractResult is the per-file result from Python /extract.
// nil Embedding means the file failed extraction on the Python side.
type PythonExtractResult struct {
	Text      string
	IDs       []ExtractedID
	Embedding []float64
}

// VerifyResult is the result from Python /verify for a single file.
type VerifyResult struct {
	Verdict           string
	SimilarityScore   float64
	SimilarityPercent string
}

// PythonAIClient is the interface for calling the CredChain Python AI service.
// Calls are serialized via an internal mutex because the Python service runs a
// single uvicorn worker and EmbeddingGemma is not thread-safe.
type PythonAIClient interface {
	Extract(ctx context.Context, files ...ExtractFile) ([]PythonExtractResult, error)
	Verify(ctx context.Context, file ExtractFile, storedEmbedding []float64) (*VerifyResult, error)
	ExtractIDs(ctx context.Context, file ExtractFile) ([]ExtractedID, error)
}

type pythonAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	mu         sync.Mutex
}

type PythonAIClientParams struct {
	Config *config.Config
}

func NewPythonAIClient(cfg *config.Config) PythonAIClient {
	return &pythonAIClient{
		baseURL:    *cfg.PythonAIBaseURL,
		apiKey:     derefOrEmpty(cfg.PythonAIAPIKey),
		httpClient: &http.Client{Timeout: time.Duration(*cfg.PythonAITimeoutSeconds) * time.Second},
	}
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (c *pythonAIClient) setAuthHeader(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
}

// resolveMIME derives the MIME type from the filename when not provided,
// using Go stdlib (mime.TypeByExtension) — no custom MIME table.
func (f ExtractFile) resolveMIME() string {
	if f.MIMEType != "" {
		return f.MIMEType
	}
	if t := mime.TypeByExtension(strings.ToLower(filepath.Ext(f.Filename))); t != "" {
		return t
	}
	return "application/octet-stream"
}

// pythonResponse is the generic envelope returned by the Python service.
type pythonResponse[T any] struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    []T                 `json:"data"`
	Errors  map[string][]string `json:"errors"`
}

type extractData struct {
	Text      string        `json:"text"`
	IDs       []ExtractedID `json:"ids"`
	Embedding []float64     `json:"embedding"`
}

type verifyData struct {
	SimilarityScore   float64 `json:"similarity_score"`
	SimilarityPercent string  `json:"similarity_percent"`
	Verdict           string  `json:"verdict"`
}

type extractIdsData struct {
	IDs []ExtractedID `json:"ids"`
}

func (c *pythonAIClient) Extract(ctx context.Context, files ...ExtractFile) ([]PythonExtractResult, error) {
	if len(files) == 0 {
		return nil, nil
	}
	body, contentType, err := buildMultipartFiles(files)
	if err != nil {
		return nil, fmt.Errorf("python extract: build multipart: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/extract", body)
	if err != nil {
		return nil, fmt.Errorf("python extract: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	c.setAuthHeader(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python extract: http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python extract: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python extract: unexpected http status %d: %s", resp.StatusCode, string(raw))
	}
	var parsed pythonResponse[*extractData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python extract: decode: %w", err)
	}
	if parsed.Code != 500100 {
		return nil, fmt.Errorf("python extract: failed (code %d, message: %s)", parsed.Code, parsed.Message)
	}
	if len(parsed.Data) != len(files) {
		return nil, fmt.Errorf("python extract: expected %d results, got %d", len(files), len(parsed.Data))
	}
	results := make([]PythonExtractResult, len(parsed.Data))
	for i, d := range parsed.Data {
		if d != nil {
			results[i] = PythonExtractResult{Text: d.Text, IDs: d.IDs, Embedding: d.Embedding}
		}
	}
	return results, nil
}

func (c *pythonAIClient) Verify(ctx context.Context, file ExtractFile, storedEmbedding []float64) (*VerifyResult, error) {
	embJSON, err := json.Marshal([][]float64{storedEmbedding})
	if err != nil {
		return nil, fmt.Errorf("python verify: marshal embeddings: %w", err)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// Set the real Content-Type on the file part. multipart.CreateFormFile
	// hardcodes application/octet-stream, which Python's validate_file rejects
	// (only pdf/jpeg/png/webp/tiff allowed) — mirror buildMultipartFiles here.
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, file.Filename))
	h.Set("Content-Type", file.resolveMIME())
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("python verify: create form file: %w", err)
	}
	if _, err := part.Write(file.Data); err != nil {
		return nil, fmt.Errorf("python verify: write file: %w", err)
	}
	if err := writer.WriteField("embeddings", string(embJSON)); err != nil {
		return nil, fmt.Errorf("python verify: write embeddings: %w", err)
	}
	writer.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/verify", body)
	if err != nil {
		return nil, fmt.Errorf("python verify: build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python verify: http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python verify: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python verify: unexpected http status %d: %s", resp.StatusCode, string(raw))
	}
	var parsed pythonResponse[*verifyData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python verify: decode: %w", err)
	}
	if parsed.Code != 500200 {
		reason := parsed.Message
		if msgs, ok := parsed.Errors["files.0"]; ok && len(msgs) > 0 {
			reason = msgs[0]
		}
		if parsed.Code == 500240 || parsed.Code == 500040 {
			return nil, fmt.Errorf("%w: %s (code %d)", ErrVerifyFileUnprocessable, reason, parsed.Code)
		}
		return nil, fmt.Errorf("python verify: failed (code %d): %s", parsed.Code, reason)
	}
	if len(parsed.Data) == 0 || parsed.Data[0] == nil {
		reason := "no result"
		if msgs, ok := parsed.Errors["files.0"]; ok && len(msgs) > 0 {
			reason = msgs[0]
		}
		return nil, fmt.Errorf("python verify: missing data: %s (code %d)", reason, parsed.Code)
	}
	d := parsed.Data[0]
	return &VerifyResult{
		Verdict:           d.Verdict,
		SimilarityScore:   d.SimilarityScore,
		SimilarityPercent: d.SimilarityPercent,
	}, nil
}

func (c *pythonAIClient) ExtractIDs(ctx context.Context, file ExtractFile) ([]ExtractedID, error) {
	body, contentType, err := buildMultipartFiles([]ExtractFile{file})
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: build multipart: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/extract-ids", body)
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	c.setAuthHeader(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python extract-ids: unexpected http status %d: %s", resp.StatusCode, string(raw))
	}
	var parsed pythonResponse[*extractIdsData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python extract-ids: decode: %w", err)
	}
	if parsed.Code != 500300 {
		return nil, fmt.Errorf("python extract-ids: failed (code %d, message: %s)", parsed.Code, parsed.Message)
	}
	if len(parsed.Data) == 0 || parsed.Data[0] == nil {
		return nil, fmt.Errorf("python extract-ids: missing data for file (code %d)", parsed.Code)
	}
	if parsed.Data[0].IDs == nil {
		return []ExtractedID{}, nil
	}
	return parsed.Data[0].IDs, nil
}

func buildMultipartFiles(files []ExtractFile) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, f := range files {
		mime := f.resolveMIME()
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, f.Filename))
		h.Set("Content-Type", mime)
		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(f.Data); err != nil {
			return nil, "", err
		}
	}
	writer.Close()
	return body, writer.FormDataContentType(), nil
}
