package http

import (
	"context"
	"net"
	"testing"
	"time"
)

// NET-02 (03/10/2026): Start respeita o host — "127.0.0.1" não pode virar todas as interfaces.
func TestStartRespeitaHost(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	s := NewEchoServer()
	go func() { _ = s.Start("127.0.0.1", port) }()
	defer func() { _ = s.Stop(context.Background()) }()

	var addr net.Addr
	for i := 0; i < 50 && addr == nil; i++ {
		addr = s.echo.ListenerAddr() // com trava (Listener direto dá corrida)
		time.Sleep(20 * time.Millisecond)
	}
	if addr == nil {
		t.Fatal("servidor não subiu")
	}
	if got := addr.(*net.TCPAddr).IP.String(); got != "127.0.0.1" {
		t.Fatalf("bind em %s, esperava 127.0.0.1", got)
	}
}
