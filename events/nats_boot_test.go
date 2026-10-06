package events

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestNatsBootPreservesEventsStream(t *testing.T) {
	bin := os.Getenv("NATS_SERVER_BIN")
	if bin == "" {
		var err error
		bin, err = exec.LookPath("nats-server")
		if err != nil {
			t.Skip("nats-server unavailable; set NATS_SERVER_BIN for integration test")
		}
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	cmd := exec.Command(bin, "-a", "127.0.0.1", "-p", fmt.Sprint(port), "-js", "-sd", t.TempDir())
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	url := fmt.Sprintf("nats://127.0.0.1:%d", port)
	var nc *nats.Conn
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		nc, err = nats.Connect(url, nats.Timeout(100*time.Millisecond))
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name: "EVENTS", Subjects: []string{"billing.>", "identity.>"},
		Description: "operator managed", Storage: jetstream.FileStorage,
		Replicas: 1, MaxAge: 48 * time.Hour, MaxMsgs: 123, MaxBytes: 1048576,
		Discard: jetstream.DiscardNew,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := s.CachedInfo().Config
	want.Subjects = append(want.Subjects, "events.>")
	for boot := 0; boot < 2; boot++ {
		env := "development"
		if boot == 1 {
			env = "production"
		}
		bus, err := NewNatsBus(url, env)
		if err != nil {
			t.Fatal(err)
		}
		bus.Close()
		info, err := s.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(info.Config, want) {
			t.Fatalf("boot %d changed operator config: got %+v, want %+v", boot, info.Config, want)
		}
	}
	if err := js.DeleteStream(ctx, "EVENTS"); err != nil {
		t.Fatal(err)
	}
	bus, err := NewNatsBus(url, "development")
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	s, err = js.Stream(ctx, "EVENTS")
	if err != nil {
		t.Fatal(err)
	}
	cfg := s.CachedInfo().Config
	if !reflect.DeepEqual(cfg.Subjects, []string{"events.>"}) || cfg.Storage != jetstream.FileStorage || cfg.Replicas != 1 || cfg.MaxAge != 24*time.Hour {
		t.Fatalf("unexpected new stream defaults: %+v", cfg)
	}
}
