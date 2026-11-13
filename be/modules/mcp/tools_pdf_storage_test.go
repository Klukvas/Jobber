package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

type pdfUploadRecord struct {
	method, path, contentType, expiry string
	body                              []byte
}
type pdfHTTPStore struct {
	mu       sync.Mutex
	objects  map[string][]byte
	requests []pdfUploadRecord
	status   int
}

func (s *pdfHTTPStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, pdfUploadRecord{method: r.Method, path: r.URL.Path, contentType: r.Header.Get("Content-Type"), expiry: r.URL.Query().Get("X-Amz-Expires"), body: body})
	if s.status != 0 {
		http.Error(w, "private S3 detail", s.status)
		return
	}
	if r.Method == http.MethodPut {
		s.objects[r.URL.Path] = body
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(s.objects[r.URL.Path])
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}
func localPDFServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		t.Skipf("sandbox blocks S3PDFStorage HTTP test TCP bind: %v", err)
	}
	require.NoError(t, err)
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	server.Start()
	t.Cleanup(server.Close)
	return server
}
func testS3PDFStorage(t *testing.T, endpoint string) S3PDFStorage {
	t.Helper()
	client, err := storage.NewS3Client(config.S3Config{Endpoint: endpoint, Bucket: "test-bucket", Region: "us-east-1", AccessKey: "fake-access", SecretKey: "fake-secret"})
	require.NoError(t, err)
	return S3PDFStorage{Client: client}
}
func pdfStorageClient(t *testing.T, f *fakeAITools, store PDFStorage) *sdk.ClientSession {
	t.Helper()
	deps := f.deps()
	deps.Storage = store
	jobs := newFakeJobs()
	return toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(deps)), userA)
}

func pdfTestEndpoint(t *testing.T, handler http.Handler, local bool) (string, *http.Client) {
	t.Helper()
	if local {
		server := localPDFServer(t, handler)
		return server.URL, server.Client()
	}
	transport := creditTransport(func(request *http.Request) (*http.Response, error) {
		copy := request.Clone(request.Context())
		if copy.Body == nil {
			copy.Body = http.NoBody
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, copy)
		result := response.Result()
		result.Request = request
		return result, nil
	})
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	return "https://s3.test", &http.Client{Transport: transport}
}
func TestS3PDFStorageToolUploadsOverwritesAndSignsGET(t *testing.T) {
	for _, mode := range []string{"in_memory_http", "local_http"} {
		t.Run(mode, func(t *testing.T) {
			store := &pdfHTTPStore{objects: map[string][]byte{}}
			endpoint, httpClient := pdfTestEndpoint(t, store, mode == "local_http")
			client := pdfStorageClient(t, newFakeAITools(), testS3PDFStorage(t, endpoint))
			path := "/test-bucket/mcp-exports/" + userA + "/" + builderA + ".pdf"
			for i := 0; i < 2; i++ {
				// Poison the existing object so repeat export must overwrite it.
				store.mu.Lock()
				store.objects[path] = []byte("old object")
				store.mu.Unlock()
				result := decodeOutput[ResumePDFOutput](t, callTool(t, client, "get_resume_pdf_link", ResumePDFInput{ResumeID: "urn:uuid:" + builderA}))
				parsed, err := url.Parse(result.URL)
				require.NoError(t, err)
				require.Equal(t, path, parsed.Path)
				require.Equal(t, "900", parsed.Query().Get("X-Amz-Expires"))
				require.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))
				response, err := httpClient.Get(result.URL)
				require.NoError(t, err)
				body, err := io.ReadAll(response.Body)
				require.NoError(t, response.Body.Close())
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)
				require.Equal(t, []byte("%PDF fake"), body)
			}
			assertPDFHTTPRequests(t, store, path)
		})
	}
}
func assertPDFHTTPRequests(t *testing.T, store *pdfHTTPStore, path string) {
	t.Helper()
	store.mu.Lock()
	defer store.mu.Unlock()
	require.Len(t, store.objects, 1)
	require.Equal(t, []byte("%PDF fake"), store.objects[path])
	require.Len(t, store.requests, 4)
	for i, request := range store.requests {
		require.Equal(t, path, request.path)
		require.Equal(t, "900", request.expiry)
		if i%2 == 0 {
			require.Equal(t, http.MethodPut, request.method)
			require.Equal(t, "application/pdf", request.contentType)
			require.Equal(t, []byte("%PDF fake"), request.body)
		} else {
			require.Equal(t, http.MethodGet, request.method)
		}
	}
}
func TestS3PDFStorageToolUploadFailureDoesNotSignDownload(t *testing.T) {
	for _, mode := range []string{"in_memory_http", "local_http"} {
		t.Run(mode, func(t *testing.T) {
			store := &pdfHTTPStore{objects: map[string][]byte{}, status: http.StatusBadGateway}
			endpoint, _ := pdfTestEndpoint(t, store, mode == "local_http")
			counted := &pdfSigningFailure{PDFStorage: testS3PDFStorage(t, endpoint)}
			result := callTool(t, pdfStorageClient(t, newFakeAITools(), counted), "get_resume_pdf_link", ResumePDFInput{ResumeID: builderA})
			assertPDFExportFailure(t, result)
			require.Zero(t, counted.signCalls)
			store.mu.Lock()
			defer store.mu.Unlock()
			require.Empty(t, store.objects)
			require.Len(t, store.requests, 1)
			require.Equal(t, http.MethodPut, store.requests[0].method)
		})
	}
}
func TestS3PDFStorageToolDownloadPresignFailure(t *testing.T) {
	for _, mode := range []string{"in_memory_http", "local_http"} {
		t.Run(mode, func(t *testing.T) {
			store := &pdfHTTPStore{objects: map[string][]byte{}}
			endpoint, _ := pdfTestEndpoint(t, store, mode == "local_http")
			counted := &pdfSigningFailure{PDFStorage: testS3PDFStorage(t, endpoint), fail: true}
			result := callTool(t, pdfStorageClient(t, newFakeAITools(), counted), "get_resume_pdf_link", ResumePDFInput{ResumeID: builderA})
			assertPDFExportFailure(t, result)
			require.Equal(t, 1, counted.signCalls)
			store.mu.Lock()
			defer store.mu.Unlock()
			require.Len(t, store.objects, 1)
			require.Len(t, store.requests, 1)
		})
	}
}
func TestS3PDFStorageInvalidEndpointPresignError(t *testing.T) {
	adapter := testS3PDFStorage(t, "://private-internal-endpoint")
	link, err := adapter.GeneratePresignedDownloadURL(t.Context(), "mcp-exports/"+userA+"/"+builderA+".pdf", "resume.pdf", pdfLinkLifetime)
	require.Error(t, err)
	require.Empty(t, link)
	require.Error(t, adapter.UploadPDF(t.Context(), "key", strings.NewReader("%PDF")))
}

// The real adapter uploads first; only the later signing failure is injected.
type pdfSigningFailure struct {
	PDFStorage
	fail      bool
	signCalls int
}

func (s *pdfSigningFailure) GeneratePresignedDownloadURL(ctx context.Context, key, name string, expiry time.Duration) (string, error) {
	s.signCalls++
	if s.fail {
		return "", errors.New("private S3 presign detail")
	}
	return s.PDFStorage.GeneratePresignedDownloadURL(ctx, key, name, expiry)
}
func assertPDFExportFailure(t *testing.T, result *sdk.CallToolResult) {
	t.Helper()
	requireToolError(t, result, "PDF export is not available")
	require.Nil(t, result.StructuredContent)
	encoded, err := json.Marshal(result.Content)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "private")
	require.NotContains(t, string(encoded), "https://")
}
