package gemini

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// dailyQuotaBody is a real 429 from the OpenAI-compatible chat endpoint
// once the free tier's 20 requests/day are used up (trimmed Help link).
const dailyQuotaBody = `[{
  "error": {
    "code": 429,
    "message": "You exceeded your current quota, please check your plan and billing details. For more information on this error, head to: https://ai.google.dev/gemini-api/docs/rate-limits. To monitor your current usage, head to: https://ai.dev/rate-limit. \n* Quota exceeded for metric: generativelanguage.googleapis.com/generate_content_free_tier_requests, limit: 20, model: gemini-3.8-flash\nPlease retry in 5h5m10.474838202s.",
    "status": "RESOURCE_EXHAUSTED",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.QuotaFailure",
        "violations": [
          {
            "quotaMetric": "generativelanguage.googleapis.com/generate_content_free_tier_requests",
            "quotaId": "GenerateRequestsPerDayPerProjectPerModel-FreeTier",
            "quotaDimensions": {"location": "global", "model": "gemini-3.8-flash"},
            "quotaValue": "20"
          }
        ]
      },
      {"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "18310s"}
    ]
  }
}
]`

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestUpstreamErrorDailyQuota(t *testing.T) {
	err := upstreamError("chat", response(http.StatusTooManyRequests, dailyQuotaBody))
	if !errors.Is(err, ErrRateLimited) || !errors.Is(err, ErrDailyQuota) {
		t.Fatalf("want ErrRateLimited and ErrDailyQuota, got %v", err)
	}
	for _, want := range []string{"limit 20", "model gemini-3.8-flash", "retry in 5h5m10s"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err, want)
		}
	}
}

func TestUpstreamErrorPerMinuteQuota(t *testing.T) {
	body := strings.ReplaceAll(dailyQuotaBody, "PerDay", "PerMinute")
	err := upstreamError("chat", response(http.StatusTooManyRequests, body))
	if !errors.Is(err, ErrRateLimited) || errors.Is(err, ErrDailyQuota) {
		t.Fatalf("want ErrRateLimited only, got %v", err)
	}
}

func TestUpstreamErrorPlainMessage(t *testing.T) {
	err := upstreamError("chat", response(http.StatusServiceUnavailable,
		`{"error":{"code":503,"message":"This model is currently experiencing high demand.","status":"UNAVAILABLE"}}`))
	if err.Error() != "gemini chat 503: This model is currently experiencing high demand." {
		t.Fatalf("got %q", err)
	}
}
