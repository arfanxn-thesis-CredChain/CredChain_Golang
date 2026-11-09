package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/infrastructure/ai/pyai"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/spf13/cobra"
)

var (
	credentialExtractionBenchmarkTimeout string
	credentialExtractionBenchmarkSizes   string
	credentialExtractionBenchmarkCount   int
	credentialExtractionBenchmarkOutput  string
	credentialExtractionBenchmarkCPUProf string
	credentialExtractionBenchmarkMemProf string
	credentialExtractionBenchmarkTrace   string
)

var credentialExtractionBenchmarkCmd = &cobra.Command{
	Use:   "credential-extraction-benchmark",
	Short: "Benchmarks the credential extraction pipeline across timeout and file size combinations",
	Long: `Runs the in-memory credential extraction pipeline (encrypt → decrypt → AI extract)
against varying timeout configurations and file sizes, collecting latency, memory, and goroutine metrics.

Pipeline: generate in-memory PDF → encrypt AES-256-GCM → decrypt AES-256-GCM → POST multipart to Python AI → parse response.

Default: 4 timeouts × 4 sizes × 3 reps = 48 pipeline calls (~$0.24-$0.28 API cost).

Examples:
  go run main.go credential-extraction-benchmark
  go run main.go credential-extraction-benchmark --output results.csv
  go run main.go credential-extraction-benchmark --timeout 30,60 --sizes 500,1000 --count 5
  go run main.go credential-extraction-benchmark --cpuprofile cpu.prof --memprofile mem.prof`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := credentialExtractionBenchmark(cmd, args); err != nil {
			log.Fatalf("benchmark failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(credentialExtractionBenchmarkCmd)

	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkTimeout, "timeout", "10,20,30,60", "Comma-separated timeout values in seconds")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkSizes, "sizes", "500,1000,2000,5000", "Comma-separated file sizes in KB")
	credentialExtractionBenchmarkCmd.Flags().IntVar(&credentialExtractionBenchmarkCount, "count", 3, "Number of concurrent goroutine runs per combination (minimum 3)")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkOutput, "output", "", "CSV output file path (empty = terminal only)")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkCPUProf, "cpuprofile", "", "Write CPU profile to file")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkMemProf, "memprofile", "", "Write memory profile to file")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkTrace, "trace", "", "Write execution trace to file")
}

func credentialExtractionBenchmarkGeneratePDF(sizeKB int) []byte {
	lorem := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. ")

	targetBytes := sizeKB * 1024

	var textBuf bytes.Buffer
	for textBuf.Len() < targetBytes {
		textBuf.Write(lorem)
	}
	textBytes := textBuf.Bytes()[:targetBytes]

	escapePDFString := func(b []byte) []byte {
		result := make([]byte, 0, len(b)+16)
		for _, ch := range b {
			switch ch {
			case '\\':
				result = append(result, '\\', '\\')
			case '(':
				result = append(result, '\\', '(')
			case ')':
				result = append(result, '\\', ')')
			default:
				result = append(result, ch)
			}
		}
		return result
	}

	maxStrLen := 60000
	var contentBuf bytes.Buffer
	contentBuf.WriteString("BT\n/F1 12 Tf\n50 750 Td\n")

	offset := 0
	for offset < len(textBytes) {
		chunkLen := len(textBytes) - offset
		if chunkLen > maxStrLen {
			chunkLen = maxStrLen
		}
		chunk := textBytes[offset : offset+chunkLen]
		escaped := escapePDFString(chunk)
		fmt.Fprintf(&contentBuf, "(%s) Tj\n0 -14 Td\n", string(escaped))
		offset += chunkLen
	}
	contentBuf.WriteString("ET\n")

	streamData := contentBuf.Bytes()

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")

	var offsets [5]int

	offsets[0] = pdf.Len()
	pdf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	offsets[1] = pdf.Len()
	pdf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	offsets[2] = pdf.Len()
	pdf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	offsets[3] = pdf.Len()
	fmt.Fprintf(&pdf, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(streamData), string(streamData))

	offsets[4] = pdf.Len()
	pdf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	xrefOffset := pdf.Len()
	pdf.WriteString("xref\n")
	fmt.Fprintf(&pdf, "0 6\n0000000000 65535 f \n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}

	fmt.Fprintf(&pdf, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	return pdf.Bytes()
}

func credentialExtractionBenchmarkRunPipeline(ctx context.Context, pdfData []byte, cfg *config.Config, client pyai.PythonAIClient) error {
	encryptedHex, err := cryptoInfra.Encrypt(pdfData, []byte(*cfg.FileEncryptionKey))
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	data, err := cryptoInfra.Decrypt(encryptedHex, []byte(*cfg.FileEncryptionKey))
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	results, err := client.Extract(ctx, pyai.ExtractFile{
		Filename: "benchmark.pdf",
		MIMEType: "application/pdf",
		Data:     data,
	})
	if err != nil {
		return fmt.Errorf("ai extract: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("ai extract: no results returned")
	}

	r := results[0]
	if r.Text == "" && r.Embedding == nil {
		return fmt.Errorf("ai extract: extraction returned empty result")
	}

	return nil
}

// credentialExtractionBenchmarkRun holds per-goroutine metrics from one pipeline execution.
type credentialExtractionBenchmarkRun struct {
	LatencyMs  float64 // wall-clock time from goroutine start to pipeline return (includes mutex wait)
	AllocMB    float64 // heap bytes allocated per operation (per-run share of batch total)
	Goroutines int     // snapshot of runtime.NumGoroutine() after this run completed
	TimedOut   bool    // true if the pipeline hit the HTTP client timeout or context deadline
}

// credentialExtractionBenchmarkResult holds aggregated metrics for one (timeout, size) combination.
type credentialExtractionBenchmarkResult struct {
	TimeoutSec     int     // HTTP client timeout in seconds for this combination
	SizeKB         int     // in-memory PDF file size in KB
	AvgMs          float64 // mean latency across successful runs (timeouts excluded)
	P50Ms          float64 // 50th percentile latency (median)
	P95Ms          float64 // 95th percentile latency (tail performance)
	P99Ms          float64 // 99th percentile latency (worst-case tail)
	MinMs          float64 // fastest successful run
	MaxMs          float64 // slowest successful run
	OpsPerSec      float64 // throughput: successful_runs / wall_clock_seconds
	AllocMBPerOp   float64 // heap bytes allocated per operation (TotalAlloc delta / count)
	GoroutinesPeak int     // peak runtime.NumGoroutine() sampled during concurrent batch execution
	MutexWaitMs    float64 // latency spread (max - min), approximates mutex queue delay from AI client serialization
	TimeoutPct     float64 // (timed_out_runs / total_runs) × 100; 100.0 → all runs timed out → ETIMEOUT sentinel
}

func credentialExtractionBenchmarkParseInts(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q: %w", p, err)
		}
		result = append(result, v)
	}
	return result, nil
}

func credentialExtractionBenchmarkComputeResult(timeoutSec, sizeKB int, runs []credentialExtractionBenchmarkRun, wallClock time.Duration, goroutinesPeak int) credentialExtractionBenchmarkResult {
	n := len(runs)

	latencies := make([]float64, n)
	var sumMs, sumAlloc float64
	var timedOutCount int
	var minMs, maxMs float64
	var firstSuccess bool

	for i, r := range runs {
		latencies[i] = r.LatencyMs
		if r.TimedOut {
			timedOutCount++
		} else {
			sumMs += r.LatencyMs
			sumAlloc += r.AllocMB
			if !firstSuccess || r.LatencyMs < minMs {
				minMs = r.LatencyMs
			}
			if !firstSuccess || r.LatencyMs > maxMs {
				maxMs = r.LatencyMs
			}
			firstSuccess = true
		}
	}

	sort.Float64s(latencies)

	successCount := n - timedOutCount
	var avgMs, p50Ms, p95Ms, p99Ms, allocMBPerOp, opsPerSec, mutexWaitMs, timeoutPct float64

	if successCount > 0 {
		avgMs = sumMs / float64(successCount)
		allocMBPerOp = sumAlloc / float64(successCount)
		opsPerSec = float64(successCount) / wallClock.Seconds()
	}

	if n > 0 {
		timeoutPct = float64(timedOutCount) / float64(n) * 100.0
	}

	p50Ms = credentialExtractionBenchmarkPercentile(latencies, 0.50)
	p95Ms = credentialExtractionBenchmarkPercentile(latencies, 0.95)
	p99Ms = credentialExtractionBenchmarkPercentile(latencies, 0.99)

	if n > 1 {
		mutexWaitMs = maxMs - minMs
		if mutexWaitMs < 0 {
			mutexWaitMs = 0
		}
	}

	return credentialExtractionBenchmarkResult{
		TimeoutSec:     timeoutSec,
		SizeKB:         sizeKB,
		AvgMs:          avgMs,
		P50Ms:          p50Ms,
		P95Ms:          p95Ms,
		P99Ms:          p99Ms,
		MinMs:          minMs,
		MaxMs:          maxMs,
		OpsPerSec:      opsPerSec,
		AllocMBPerOp:   allocMBPerOp,
		GoroutinesPeak: goroutinesPeak,
		MutexWaitMs:    mutexWaitMs,
		TimeoutPct:     timeoutPct,
	}
}

func credentialExtractionBenchmarkPercentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func credentialExtractionBenchmarkWriteTerminal(results []credentialExtractionBenchmarkResult) {
	fmt.Println()
	fmt.Println("┌───────────┬─────────┬────────┬────────┬────────┬────────┬────────┬────────┬───────────┬───────────────┬───────────────┬──────────────┬───────────────┐")
	fmt.Println("│ Timeout(s)│ Size(KB)│ Avg(ms)│ P50(ms)│ P95(ms)│ P99(ms)│ Min(ms)│ Max(ms)│ Ops/sec   │ AllocMB/op    │ Gorout. peak  │ MutexWait(ms)│ Timeout %     │")
	fmt.Println("├───────────┼─────────┼────────┼────────┼────────┼────────┼────────┼────────┼───────────┼───────────────┼───────────────┼──────────────┼───────────────┤")

	for _, r := range results {
		row := credentialExtractionBenchmarkFormatRow(r)
		fmt.Println(row)
	}
	fmt.Println("└───────────┴─────────┴────────┴────────┴────────┴────────┴────────┴────────┴───────────┴───────────────┴───────────────┴──────────────┴───────────────┘")
	fmt.Println()
}

func credentialExtractionBenchmarkFormatRow(r credentialExtractionBenchmarkResult) string {
	formatFloat := func(v float64, width int, prec int) string {
		s := fmt.Sprintf("%*.*f", width, prec, v)
		if len(s) > width {
			s = s[:width-1] + ">"
		}
		return s
	}

	formatStr := func(s string, width int) string {
		if len(s) > width {
			return s[:width]
		}
		return fmt.Sprintf("%*s", width, s)
	}

	timeoutStr := fmt.Sprintf("%d", r.TimeoutSec)
	sizeStr := fmt.Sprintf("%d", r.SizeKB)

	if r.TimeoutPct == 100.0 {
		timeoutLabel := "ETIMEOUT"
		return fmt.Sprintf("│ %-9s │ %-7s │ %-6s │ %-6s │ %-6s │ %-6s │ %-6s │ %-6s │ %-9s │ %-13s │ %-13s │ %-12s │ %-13s │",
			formatStr(timeoutStr, 9),
			formatStr(sizeStr, 7),
			formatStr(timeoutLabel, 6), formatStr(timeoutLabel, 6), formatStr(timeoutLabel, 6), formatStr(timeoutLabel, 6),
			formatStr(timeoutLabel, 6), formatStr(timeoutLabel, 6),
			formatStr("0.00", 9),
			formatStr("0.0", 13),
			formatStr(fmt.Sprintf("%d", r.GoroutinesPeak), 13),
			formatStr("0", 12),
			formatStr("100.0", 13),
		)
	}

	return fmt.Sprintf("│ %-9s │ %-7s │ %-6s │ %-6s │ %-6s │ %-6s │ %-6s │ %-6s │ %-9s │ %-13s │ %-13s │ %-12s │ %-13s │",
		formatStr(timeoutStr, 9),
		formatStr(sizeStr, 7),
		formatFloat(r.AvgMs, 6, 2), formatFloat(r.P50Ms, 6, 2), formatFloat(r.P95Ms, 6, 2), formatFloat(r.P99Ms, 6, 2),
		formatFloat(r.MinMs, 6, 2), formatFloat(r.MaxMs, 6, 2),
		formatFloat(r.OpsPerSec, 9, 2),
		formatFloat(r.AllocMBPerOp, 13, 1),
		formatStr(fmt.Sprintf("%d", r.GoroutinesPeak), 13),
		formatFloat(r.MutexWaitMs, 12, 2),
		formatFloat(r.TimeoutPct, 13, 1),
	)
}

func credentialExtractionBenchmarkWriteCSV(w *csv.Writer, results []credentialExtractionBenchmarkResult) error {
	header := []string{"timeout_sec", "size_kb", "avg_ms", "p50_ms", "p95_ms", "p99_ms", "min_ms", "max_ms", "ops_per_sec", "alloc_mb_per_op", "goroutines_peak", "mutex_wait_ms", "timeout_hit_pct"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range results {
		row := credentialExtractionBenchmarkCSVRow(r)
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func credentialExtractionBenchmarkCSVRow(r credentialExtractionBenchmarkResult) []string {
	formatFloat := func(v float64) string {
		return strconv.FormatFloat(v, 'f', 2, 64)
	}

	timeout := fmt.Sprintf("%d", r.TimeoutSec)
	size := fmt.Sprintf("%d", r.SizeKB)

	if r.TimeoutPct == 100.0 {
		return []string{timeout, size, "ETIMEOUT", "ETIMEOUT", "ETIMEOUT", "ETIMEOUT", "ETIMEOUT", "ETIMEOUT", "0.00", "0.0", fmt.Sprintf("%d", r.GoroutinesPeak), "0", "100.0"}
	}

	return []string{
		timeout, size,
		formatFloat(r.AvgMs), formatFloat(r.P50Ms), formatFloat(r.P95Ms), formatFloat(r.P99Ms),
		formatFloat(r.MinMs), formatFloat(r.MaxMs),
		formatFloat(r.OpsPerSec), formatFloat(r.AllocMBPerOp),
		fmt.Sprintf("%d", r.GoroutinesPeak),
		formatFloat(r.MutexWaitMs), formatFloat(r.TimeoutPct),
	}
}

func credentialExtractionBenchmarkPrintColumnDescriptions() {
	fmt.Println()
	fmt.Println("Column descriptions")
	fmt.Println("===================")
	fmt.Println()
	fmt.Println("Timeout(s)    HTTP client timeout for this combination in seconds.")
	fmt.Println()
	fmt.Println("Size(KB)      In memory PDF file size sent through the pipeline in")
	fmt.Println("              kilobytes.")
	fmt.Println()
	fmt.Println("Avg(ms)       Mean latency across successful runs only. Timed out")
	fmt.Println("              runs are excluded from the average.")
	fmt.Println()
	fmt.Println("P50(ms)       Median latency. 50% of runs complete faster than this")
	fmt.Println("              and 50% complete slower.")
	fmt.Println()
	fmt.Println("P95(ms)       95th percentile latency. Only 5% of runs are slower.")
	fmt.Println("              Shows tail performance near the worst case.")
	fmt.Println()
	fmt.Println("P99(ms)       99th percentile latency. The worst remaining 1% after")
	fmt.Println("              trimming the single worst outlier.")
	fmt.Println()
	fmt.Println("Min(ms)       Fastest successful pipeline run in this batch.")
	fmt.Println()
	fmt.Println("Max(ms)       Slowest successful pipeline run in this batch. Timeouts")
	fmt.Println("              are excluded from this column.")
	fmt.Println()
	fmt.Println("Ops/sec       Throughput measured as successful pipelines completed")
	fmt.Println("              per wall clock second.")
	fmt.Println()
	fmt.Println("AllocMB/op    Heap bytes allocated per operation averaged across all")
	fmt.Println("              goroutines.")
	fmt.Println()
	fmt.Println("Gorout. peak  Highest observed goroutine count during this batch")
	fmt.Println("              sampled every 10 milliseconds.")
	fmt.Println()
	fmt.Println("MutexWait(ms) Latency spread computed as the difference between the")
	fmt.Println("              slowest and fastest run. Since goroutines serialize on")
	fmt.Println("              a mutex in the AI client, earlier goroutines grab the")
	fmt.Println("              lock first with low latency while later ones queue with")
	fmt.Println("              higher latency. The spread measures how long the last")
	fmt.Println("              goroutine waited compared to the first.")
	fmt.Println()
	fmt.Println("Timeout %     Fraction of runs that hit the timeout deadline. When")
	fmt.Println("              this is 100 percent, every run in the batch timed out")
	fmt.Println("              and the latency columns show ETIMEOUT.")
	fmt.Println()
}

type benchmarkPythonAIClient struct {
	baseURL string
	apiKey  string
	timeout time.Duration
	mu      sync.Mutex
}

func (c *benchmarkPythonAIClient) Extract(ctx context.Context, files ...pyai.ExtractFile) ([]pyai.PythonExtractResult, error) {
	if len(files) == 0 {
		return nil, nil
	}

	body, contentType, err := benchmarkBuildMultipart(files)
	if err != nil {
		return nil, fmt.Errorf("python extract: build multipart: %w", err)
	}

	httpClient := &http.Client{Timeout: c.timeout}

	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/extract", body)
	if err != nil {
		return nil, fmt.Errorf("python extract: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	resp, err := httpClient.Do(req)
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

	type extractData struct {
		Text      string             `json:"text"`
		IDs       []pyai.ExtractedID `json:"ids"`
		Embedding []float64          `json:"embedding"`
	}
	type pythonResponse struct {
		Code    int                 `json:"code"`
		Message string              `json:"message"`
		Data    []*extractData      `json:"data"`
		Errors  map[string][]string `json:"errors"`
	}

	var parsed pythonResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python extract: decode: %w", err)
	}

	if parsed.Code != 500100 {
		return nil, fmt.Errorf("python extract: failed (code %d, message: %s)", parsed.Code, parsed.Message)
	}
	if len(parsed.Data) != len(files) {
		return nil, fmt.Errorf("python extract: expected %d results, got %d", len(files), len(parsed.Data))
	}

	results := make([]pyai.PythonExtractResult, len(parsed.Data))
	for i, d := range parsed.Data {
		if d != nil {
			results[i] = pyai.PythonExtractResult{Text: d.Text, IDs: d.IDs, Embedding: d.Embedding}
		}
	}
	return results, nil
}

func (c *benchmarkPythonAIClient) Verify(ctx context.Context, file pyai.ExtractFile, storedEmbedding []float64) (*pyai.VerifyResult, error) {
	return nil, fmt.Errorf("verify not available in benchmark client")
}

func (c *benchmarkPythonAIClient) ExtractIDs(ctx context.Context, file pyai.ExtractFile) ([]pyai.ExtractedID, error) {
	return nil, fmt.Errorf("extract-ids not available in benchmark client")
}

func benchmarkBuildMultipart(files []pyai.ExtractFile) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, f := range files {
		mimeType := f.MIMEType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, f.Filename))
		h.Set("Content-Type", mimeType)
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

func credentialExtractionBenchmark(cmd *cobra.Command, args []string) error {
	timeouts, err := credentialExtractionBenchmarkParseInts(credentialExtractionBenchmarkTimeout)
	if err != nil {
		return fmt.Errorf("invalid --timeout: %w", err)
	}

	sizes, err := credentialExtractionBenchmarkParseInts(credentialExtractionBenchmarkSizes)
	if err != nil {
		return fmt.Errorf("invalid --sizes: %w", err)
	}

	count := credentialExtractionBenchmarkCount
	if count < 1 {
		return fmt.Errorf("--count must be >= 1, got %d", count)
	}

	cfg := ConfigFromCmd(cmd)
	if cfg.FileEncryptionKey == nil {
		return fmt.Errorf("FILE_ENCRYPTION_KEY is not set")
	}

	log.Printf("Benchmark: %d timeouts × %d sizes × %d reps = %d pipeline calls",
		len(timeouts), len(sizes), count, len(timeouts)*len(sizes)*count)
	log.Printf("Timeouts (s): %v", timeouts)
	log.Printf("Sizes (KB):   %v", sizes)
	log.Printf("Runs/comb:    %d (concurrent goroutines)", count)
	log.Println()

	credentialExtractionBenchmarkPrintColumnDescriptions()

	if credentialExtractionBenchmarkCPUProf != "" {
		f, err := os.Create(credentialExtractionBenchmarkCPUProf)
		if err != nil {
			return fmt.Errorf("create cpu profile: %w", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("start cpu profile: %w", err)
		}
		defer pprof.StopCPUProfile()
		log.Printf("CPU profiling enabled → %s", credentialExtractionBenchmarkCPUProf)
	}

	if credentialExtractionBenchmarkTrace != "" {
		f, err := os.Create(credentialExtractionBenchmarkTrace)
		if err != nil {
			return fmt.Errorf("create trace: %w", err)
		}
		defer f.Close()
		if err := trace.Start(f); err != nil {
			return fmt.Errorf("start trace: %w", err)
		}
		defer trace.Stop()
		log.Printf("Execution tracing enabled → %s", credentialExtractionBenchmarkTrace)
	}

	var allResults []credentialExtractionBenchmarkResult

	for _, timeoutSec := range timeouts {
		for _, sizeKB := range sizes {
			log.Printf("Running: timeout=%ds size=%dKB reps=%d", timeoutSec, sizeKB, count)

			pdfData := credentialExtractionBenchmarkGeneratePDF(sizeKB)

			timeoutDuration := time.Duration(timeoutSec) * time.Second
			client := &benchmarkPythonAIClient{
				baseURL: *cfg.PythonAIBaseURL,
				apiKey:  derefBenchmark(cfg.PythonAIAPIKey),
				timeout: timeoutDuration,
			}

			runs, goroutinesPeak, wallClock := credentialExtractionBenchmarkRunConcurrent(context.Background(), pdfData, cfg, client, count)

			result := credentialExtractionBenchmarkComputeResult(timeoutSec, sizeKB, runs, wallClock, goroutinesPeak)
			allResults = append(allResults, result)

			credentialExtractionBenchmarkWriteTerminal([]credentialExtractionBenchmarkResult{result})
		}
	}

	credentialExtractionBenchmarkWriteTerminal(allResults)

	if credentialExtractionBenchmarkMemProf != "" {
		f, err := os.Create(credentialExtractionBenchmarkMemProf)
		if err != nil {
			return fmt.Errorf("create mem profile: %w", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("write mem profile: %w", err)
		}
		log.Printf("Memory profile written → %s", credentialExtractionBenchmarkMemProf)
	}

	if credentialExtractionBenchmarkOutput != "" {
		f, err := os.Create(credentialExtractionBenchmarkOutput)
		if err != nil {
			return fmt.Errorf("create csv output: %w", err)
		}
		defer f.Close()
		w := csv.NewWriter(f)
		defer w.Flush()
		if err := credentialExtractionBenchmarkWriteCSV(w, allResults); err != nil {
			return fmt.Errorf("write csv: %w", err)
		}
		log.Printf("CSV written → %s", credentialExtractionBenchmarkOutput)
	}

	log.Println("Benchmark complete.")
	return nil
}

func derefBenchmark(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func credentialExtractionBenchmarkRunConcurrent(ctx context.Context, pdfData []byte, cfg *config.Config, client pyai.PythonAIClient, count int) ([]credentialExtractionBenchmarkRun, int, time.Duration) {
	runs := make([]credentialExtractionBenchmarkRun, count)

	peakGoroutines := 0
	var peakMu sync.Mutex

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n := runtime.NumGoroutine()
				peakMu.Lock()
				if n > peakGoroutines {
					peakGoroutines = n
				}
				peakMu.Unlock()
			case <-done:
				return
			}
		}
	}()

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	wallStart := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			start := time.Now()
			err := credentialExtractionBenchmarkRunPipeline(ctx, pdfData, cfg, client)
			elapsed := time.Since(start)

			var timedOut bool
			if err != nil {
				timedOut = errors.Is(err, context.DeadlineExceeded)
				if !timedOut {
					var urlErr *url.Error
					if errors.As(err, &urlErr) && urlErr.Timeout() {
						timedOut = true
					}
				}
			}

			runs[idx] = credentialExtractionBenchmarkRun{
				LatencyMs:  float64(elapsed.Microseconds()) / 1000.0,
				AllocMB:    0,
				Goroutines: runtime.NumGoroutine(),
				TimedOut:   timedOut,
			}
		}(i)
	}

	wg.Wait()
	wallClock := time.Since(wallStart)

	runtime.ReadMemStats(&memAfter)
	close(done)

	allocDelta := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / (1024 * 1024)
	perRunAlloc := allocDelta / float64(count)

	for i := range runs {
		// All goroutines process the same PDF pipeline; per-run alloc is averaged
		// across all runs since individual heap tracking is not possible with
		// shared http.Transport buffers.
		runs[i].AllocMB = perRunAlloc
	}

	return runs, peakGoroutines, wallClock
}
