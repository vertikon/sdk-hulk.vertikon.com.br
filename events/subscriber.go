package events

// Subscriber fornece métodos auxiliares para inscrição em eventos.
type Subscriber struct {
	bus Bus
}

// NewSubscriber cria um novo Subscriber.
func NewSubscriber(bus Bus) *Subscriber {
	return &Subscriber{bus: bus}
}

// SubscribeToTopic inscreve-se em um tópico específico.
func (s *Subscriber) SubscribeToTopic(topic string, handler Handler) error {
	return s.bus.Subscribe(topic, handler)
}

// SubscribeToQueue inscreve-se em uma fila para load balancing.
func (s *Subscriber) SubscribeToQueue(topic, queue string, handler Handler) error {
	return s.bus.QueueSubscribe(topic, queue, handler)
}
