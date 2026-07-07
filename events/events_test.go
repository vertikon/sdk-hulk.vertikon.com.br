package events

import (
	"sync"
	"testing"
	"time"
)

// MockMessage implementa a interface Message para testes
type MockMessage struct {
	id      string
	topic   string
	payload []byte
	acked   bool
	nacked  bool
	mu      sync.Mutex
}

func (m *MockMessage) ID() string {
	return m.id
}

func (m *MockMessage) Topic() string {
	return m.topic
}

func (m *MockMessage) Payload() []byte {
	return m.payload
}

func (m *MockMessage) Ack() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acked = true
	return nil
}

func (m *MockMessage) Nak() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nacked = true
	return nil
}

func (m *MockMessage) Nack() error {
	return m.Nak()
}

func (m *MockMessage) IsAcked() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.acked
}

// MockEventBus implementa EventBus para testes
type MockEventBus struct {
	published   []PublishEvent
	subscribers map[string][]Handler
	mu          sync.RWMutex
}

type PublishEvent struct {
	Topic   string
	Payload interface{}
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		published:   make([]PublishEvent, 0),
		subscribers: make(map[string][]Handler),
	}
}

func (m *MockEventBus) Publish(topic string, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.published = append(m.published, PublishEvent{
		Topic:   topic,
		Payload: payload,
	})

	// Notificar subscribers
	if handlers, ok := m.subscribers[topic]; ok {
		msg := &MockMessage{
			id:      "test-msg-id",
			topic:   topic,
			payload: []byte("test-payload"),
		}

		for _, handler := range handlers {
			go handler(msg)
		}
	}

	return nil
}

func (m *MockEventBus) Subscribe(topic string, handler Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers[topic] == nil {
		m.subscribers[topic] = make([]Handler, 0)
	}

	m.subscribers[topic] = append(m.subscribers[topic], handler)
	return nil
}

func (m *MockEventBus) QueueSubscribe(topic, queue string, handler Handler) error {
	// Para testes, QueueSubscribe funciona igual a Subscribe
	return m.Subscribe(topic, handler)
}

func (m *MockEventBus) GetPublished() []PublishEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.published
}

// Testes

func TestMockMessage_Interface(t *testing.T) {
	msg := &MockMessage{
		id:      "msg-123",
		topic:   "test.topic",
		payload: []byte("test payload"),
	}

	if msg.ID() != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got '%s'", msg.ID())
	}

	if msg.Topic() != "test.topic" {
		t.Errorf("Expected topic 'test.topic', got '%s'", msg.Topic())
	}

	if string(msg.Payload()) != "test payload" {
		t.Errorf("Expected payload 'test payload', got '%s'", string(msg.Payload()))
	}

	if msg.IsAcked() {
		t.Error("Message should not be acked initially")
	}

	if err := msg.Ack(); err != nil {
		t.Errorf("Ack() failed: %v", err)
	}

	if !msg.IsAcked() {
		t.Error("Message should be acked after Ack()")
	}
}

func TestMockMessage_Nack(t *testing.T) {
	msg := &MockMessage{
		id:      "msg-456",
		topic:   "test.nack",
		payload: []byte("nack test"),
	}

	if err := msg.Nack(); err != nil {
		t.Errorf("Nack() failed: %v", err)
	}

	msg.mu.Lock()
	nacked := msg.nacked
	msg.mu.Unlock()

	if !nacked {
		t.Error("Message should be nacked after Nack()")
	}
}

func TestMockEventBus_PublishSubscribe(t *testing.T) {
	bus := NewMockEventBus()
	topic := "test.event"

	received := make(chan Message, 1)

	// Subscribe
	err := bus.Subscribe(topic, func(msg Message) error {
		received <- msg
		return msg.Ack()
	})

	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Publish
	payload := map[string]string{"key": "value"}
	err = bus.Publish(topic, payload)

	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Verificar que foi publicado
	published := bus.GetPublished()
	if len(published) != 1 {
		t.Fatalf("Expected 1 published event, got %d", len(published))
	}

	if published[0].Topic != topic {
		t.Errorf("Expected topic '%s', got '%s'", topic, published[0].Topic)
	}

	// Verificar que subscriber recebeu
	select {
	case msg := <-received:
		if msg.Topic() != topic {
			t.Errorf("Expected topic '%s', got '%s'", topic, msg.Topic())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber did not receive message")
	}
}

func TestMockEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewMockEventBus()
	topic := "test.multi"

	received1 := make(chan Message, 1)
	received2 := make(chan Message, 1)

	// Subscribe 1
	bus.Subscribe(topic, func(msg Message) error {
		received1 <- msg
		return msg.Ack()
	})

	// Subscribe 2
	bus.Subscribe(topic, func(msg Message) error {
		received2 <- msg
		return msg.Ack()
	})

	// Publish
	bus.Publish(topic, "test")

	// Ambos devem receber
	timeout := time.After(100 * time.Millisecond)
	count := 0

	for count < 2 {
		select {
		case <-received1:
			count++
		case <-received2:
			count++
		case <-timeout:
			t.Fatalf("Expected 2 subscribers to receive, got %d", count)
		}
	}
}

func TestBus_Interface(t *testing.T) {
	// Teste básico para garantir que Bus interface existe
	var _ Bus = (*MockEventBus)(nil)
}

func TestHandler_Execution(t *testing.T) {
	executed := false

	handler := func(msg Message) error {
		executed = true
		return nil
	}

	msg := &MockMessage{
		id:      "test",
		topic:   "test",
		payload: []byte("test"),
	}

	err := handler(msg)

	if err != nil {
		t.Errorf("Handler failed: %v", err)
	}

	if !executed {
		t.Error("Handler was not executed")
	}
}

func TestMessage_ConcurrentAck(t *testing.T) {
	msg := &MockMessage{
		id:      "concurrent",
		topic:   "test",
		payload: []byte("test"),
	}

	var wg sync.WaitGroup

	// Tentar ack concorrente
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg.Ack()
		}()
	}

	wg.Wait()

	if !msg.IsAcked() {
		t.Error("Message should be acked")
	}
}

func TestEventBus_PublishWithoutSubscribers(t *testing.T) {
	bus := NewMockEventBus()

	err := bus.Publish("no.subscribers", "test")

	if err != nil {
		t.Errorf("Publish without subscribers should not error: %v", err)
	}

	published := bus.GetPublished()
	if len(published) != 1 {
		t.Errorf("Expected 1 published event, got %d", len(published))
	}
}

func TestEventBus_SubscribeMultipleTopics(t *testing.T) {
	bus := NewMockEventBus()

	received := make(map[string]int)
	var mu sync.Mutex

	handler := func(topic string) Handler {
		return func(msg Message) error {
			mu.Lock()
			received[topic]++
			mu.Unlock()
			return msg.Ack()
		}
	}

	topics := []string{"topic1", "topic2", "topic3"}

	for _, topic := range topics {
		bus.Subscribe(topic, handler(topic))
	}

	for _, topic := range topics {
		bus.Publish(topic, "test")
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	for _, topic := range topics {
		if received[topic] != 1 {
			t.Errorf("Expected topic '%s' to receive 1 message, got %d", topic, received[topic])
		}
	}
}

// Benchmark tests

func BenchmarkEventBus_Publish(b *testing.B) {
	bus := NewMockEventBus()
	payload := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish("bench.topic", payload)
	}
}

func BenchmarkEventBus_Subscribe(b *testing.B) {
	bus := NewMockEventBus()
	handler := func(msg Message) error {
		return msg.Ack()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Subscribe("bench.topic", handler)
	}
}

func BenchmarkMessage_Ack(b *testing.B) {
	msg := &MockMessage{
		id:      "bench",
		topic:   "bench",
		payload: []byte("bench"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.Ack()
	}
}

// Publisher Tests

func TestPublisher_New(t *testing.T) {
	bus := NewMockEventBus()
	pub := NewPublisher(bus)

	if pub == nil {
		t.Fatal("NewPublisher returned nil")
	}
}

func TestPublisher_PublishJSON(t *testing.T) {
	bus := NewMockEventBus()
	pub := NewPublisher(bus)

	payload := map[string]string{"key": "value"}
	err := pub.PublishJSON("test.json", payload)

	if err != nil {
		t.Errorf("PublishJSON failed: %v", err)
	}

	published := bus.GetPublished()
	if len(published) != 1 {
		t.Fatalf("Expected 1 published event, got %d", len(published))
	}

	if published[0].Topic != "test.json" {
		t.Errorf("Expected topic 'test.json', got '%s'", published[0].Topic)
	}

	// Check payload
	if p, ok := published[0].Payload.(map[string]string); ok {
		if p["key"] != "value" {
			t.Errorf("Expected payload value 'value', got '%s'", p["key"])
		}
	} else {
		t.Error("Payload type mismatch")
	}
}

func TestPublisher_PublishBytes(t *testing.T) {
	bus := NewMockEventBus()
	pub := NewPublisher(bus)

	payload := []byte("test bytes")
	err := pub.PublishBytes("test.bytes", payload)

	if err != nil {
		t.Errorf("PublishBytes failed: %v", err)
	}

	published := bus.GetPublished()
	if len(published) != 1 {
		t.Fatalf("Expected 1 published event, got %d", len(published))
	}

	if published[0].Topic != "test.bytes" {
		t.Errorf("Expected topic 'test.bytes', got '%s'", published[0].Topic)
	}

	if string(published[0].Payload.([]byte)) != "test bytes" {
		t.Errorf("Expected payload 'test bytes', got '%s'", string(published[0].Payload.([]byte)))
	}
}

// Subscriber Tests

func TestSubscriber_New(t *testing.T) {
	bus := NewMockEventBus()
	sub := NewSubscriber(bus)

	if sub == nil {
		t.Fatal("NewSubscriber returned nil")
	}
}

func TestSubscriber_SubscribeToTopic(t *testing.T) {
	bus := NewMockEventBus()
	sub := NewSubscriber(bus)

	received := false
	handler := func(msg Message) error {
		received = true
		return nil
	}

	err := sub.SubscribeToTopic("test.sub", handler)
	if err != nil {
		t.Errorf("SubscribeToTopic failed: %v", err)
	}

	// Verify subscription by publishing
	bus.Publish("test.sub", "test")
	time.Sleep(50 * time.Millisecond)

	if !received {
		t.Error("Handler not called after SubscribeToTopic")
	}
}

func TestSubscriber_SubscribeToQueue(t *testing.T) {
	bus := NewMockEventBus()
	sub := NewSubscriber(bus)

	received := false
	handler := func(msg Message) error {
		received = true
		return nil
	}

	err := sub.SubscribeToQueue("test.queue", "queue-group", handler)
	if err != nil {
		t.Errorf("SubscribeToQueue failed: %v", err)
	}

	// Verify subscription by publishing
	bus.Publish("test.queue", "test")
	time.Sleep(50 * time.Millisecond)

	if !received {
		t.Error("Handler not called after SubscribeToQueue")
	}
}
