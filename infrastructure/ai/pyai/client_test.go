package pyai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(baseURL, apiKey string) *pythonAIClient {
	return &pythonAIClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func TestExtract_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500100,
			"message": "ok",
			"data": []map[string]any{
				{
					"text": "extracted-text",
					"ids": []map[string]string{
						{"type": "id", "value": "123"},
					},
					"embedding": []float64{0.1, 0.2},
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	results, err := client.Extract(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "extracted-text", results[0].Text)
	require.Len(t, results[0].IDs, 1)
	assert.Equal(t, "id", results[0].IDs[0].Type)
	assert.Equal(t, "123", results[0].IDs[0].Value)
	assert.Equal(t, []float64{0.1, 0.2}, results[0].Embedding)
}

func TestExtract_AllFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500150,
			"message": "all failed",
			"data":    []any{},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	_, err := client.Extract(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500150")
}

func TestExtract_SetsAuthHeader(t *testing.T) {
	var capturedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500100,
			"message": "ok",
			"data": []map[string]any{
				{"text": "ok", "ids": []any{}, "embedding": []float64{0.1}},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "test-key")
	_, err := client.Extract(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	require.NoError(t, err)
	assert.Equal(t, "test-key", capturedHeaders.Get("X-Api-Key"))
}

func TestVerify_RequestShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		_, params, err := mime.ParseMediaType(contentType)
		require.NoError(t, err)

		reader := multipart.NewReader(r.Body, params["boundary"])
		var embeddingsFound bool
		var filesFound bool

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			switch part.FormName() {
			case "embeddings":
				embeddingsFound = true
				b, _ := io.ReadAll(part)
				assert.JSONEq(t, `[[0.1,0.2]]`, string(b))
			case "files":
				filesFound = true
				// Must carry the real MIME (test.pdf -> application/pdf), not
				// application/octet-stream, or Python's validate_file rejects it.
				assert.Equal(t, "application/pdf", part.Header.Get("Content-Type"))
			}
		}

		assert.True(t, embeddingsFound, "embeddings form field should be present")
		assert.True(t, filesFound, "files form file should be present")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500200,
			"message": "ok",
			"data": []map[string]any{
				{
					"similarity_score":   0.95,
					"similarity_percent": "95.0%",
					"verdict":            "tampered",
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	result, err := client.Verify(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test-content"),
	}, []float64{0.1, 0.2})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "tampered", result.Verdict)
	assert.Equal(t, 0.95, result.SimilarityScore)
	assert.Equal(t, "95.0%", result.SimilarityPercent)
}

func TestExtractIDs_EmptyReturnsSlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500300,
			"message": "ok",
			"data": []map[string]any{
				{"ids": []any{}},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	ids, err := client.ExtractIDs(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	require.NoError(t, err)
	assert.NotNil(t, ids)
	assert.Empty(t, ids)
}

func TestResolveMIME_FromFilename(t *testing.T) {
	f := ExtractFile{Filename: "test.pdf"}
	assert.Equal(t, "application/pdf", f.resolveMIME())
}

func TestResolveMIME_Explicit(t *testing.T) {
	f := ExtractFile{Filename: "test.pdf", MIMEType: "image/jpeg"}
	assert.Equal(t, "image/jpeg", f.resolveMIME())
}

func TestExtract_HTTPErrorReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500010,
			"message": "Invalid or missing API key",
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "bad-key")
	_, err := client.Extract(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestVerify_TechnicalErrorDoesNotReturnUnprocessable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500150,
			"message": "Gemini extraction failed",
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	_, err := client.Verify(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	}, []float64{0.1, 0.2})
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrVerifyFileUnprocessable), "HTTP 500 must not return ErrVerifyFileUnprocessable")
}

func TestVerify_MissingSuccessDataReturnsTechnicalError(t *testing.T) {
	for _, data := range []string{`[]`, `[null]`} {
		t.Run(data, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"code":500200,"data":%s}`, data)
			}))
			defer srv.Close()

			client := newTestClient(srv.URL, "")
			result, err := client.Verify(context.Background(), ExtractFile{
				Filename: "test.pdf",
				Data:     []byte("test"),
			}, []float64{0.1, 0.2})
			require.Error(t, err)
			assert.Nil(t, result)
			assert.False(t, errors.Is(err, ErrVerifyFileUnprocessable))
		})
	}
}

func TestVerify_DocumentErrorReturnsUnprocessable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500240,
			"message": "No readable text",
			"data":    []any{nil},
			"errors": map[string][]string{
				"files.0": {"No text extracted from document"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	_, err := client.Verify(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	}, []float64{0.1, 0.2})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrVerifyFileUnprocessable))
}

func TestExtractIDs_ErrorEnvelopeReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500150,
			"message": "Gemini failed",
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	_, err := client.ExtractIDs(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	assert.Error(t, err, "Technical error in ExtractIDs must return an error, not empty slice")
}

func TestExtractIDs_HTTP200ErrorEnvelopeReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    500350,
			"message": "OCR failed",
			"data":    []any{nil},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "")
	_, err := client.ExtractIDs(context.Background(), ExtractFile{
		Filename: "test.pdf",
		Data:     []byte("test"),
	})
	assert.Error(t, err, "Error envelope in ExtractIDs must return an error, not empty slice")
}
