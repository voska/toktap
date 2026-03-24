package influx

import (
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/voska/toktap/internal/proxy"
	"github.com/voska/toktap/internal/sse"
)

type writeAPI interface {
	WriteRecord(line string)
	Flush()
	Errors() <-chan error
}

type Writer struct {
	api    writeAPI
	client influxdb2.Client
}

func NewWriter(url, token, org, bucket string) *Writer {
	client := influxdb2.NewClientWithOptions(url, token,
		influxdb2.DefaultOptions().SetBatchSize(20).SetFlushInterval(1000))
	writeAPI := client.WriteAPI(org, bucket)

	go func() {
		for err := range writeAPI.Errors() {
			log.Printf("influxdb write error: %v", err)
		}
	}()

	return &Writer{api: writeAPI, client: client}
}

func (w *Writer) WriteUsage(usage sse.UsageData, meta proxy.Metadata, costUSD float64, responseTimeMs int64, statusCode int, ts time.Time, requestID string) {
	line := fmt.Sprintf(
		"llm_usage,provider=%s,model=%s,device=%s,harness=%s,auth_type=%s,request_id=%s input_tokens=%di,output_tokens=%di,cache_read_tokens=%di,cache_creation_tokens=%di,cost_usd=%.6f,response_time_ms=%di,status_code=%di %d",
		escape(meta.Provider),
		escape(usage.Model),
		escape(meta.Device),
		escape(meta.Harness),
		escape(meta.AuthType),
		escape(requestID),
		usage.InputTokens,
		usage.OutputTokens,
		usage.CacheReadTokens,
		usage.CacheCreationTokens,
		costUSD,
		responseTimeMs,
		statusCode,
		ts.UnixNano(),
	)
	w.api.WriteRecord(line)
}

func (w *Writer) Close() {
	if w.api != nil {
		if flusher, ok := w.api.(api.WriteAPI); ok {
			flusher.Flush()
		}
	}
	if w.client != nil {
		w.client.Close()
	}
}

func escape(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ',', ' ', '=':
			result = append(result, '\\')
		}
		result = append(result, s[i])
	}
	if len(result) == 0 {
		return "unknown"
	}
	return string(result)
}
