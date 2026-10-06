package hulk

import "testing"

func TestCollectorURLFromEnv(t *testing.T) {
	for _, tc := range []struct{ name, collector, endpoint, want string }{
		{"absent", "", "", ""},
		{"blank", "  ", "\t", ""},
		{"collector", " http://collector:4317 ", "other:4317", "collector:4317"},
		{"endpoint", "", "https://otel:4317", "otel:4317"},
		{"blank collector fallback", " ", "otel:4317", "otel:4317"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTEL_COLLECTOR_URL", tc.collector)
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tc.endpoint)
			if got := collectorURLFromEnv(); got != tc.want {
				t.Fatalf("collector endpoint = %q, want %q", got, tc.want)
			}
		})
	}
}
