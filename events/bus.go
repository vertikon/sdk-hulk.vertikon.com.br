package events

// Message representa um evento genérico no sistema.
type Message interface {
	ID() string
	Topic() string
	Payload() []byte
	Ack() error
	Nak() error
}

// Handler é a função que processa eventos recebidos.
type Handler func(msg Message) error

// Bus define como os módulos interagem com o NATS JetStream.
type Bus interface {
	// Publish envia um evento para o ecossistema.
	Publish(topic string, payload interface{}) error

	// Subscribe escuta eventos de um tópico.
	Subscribe(topic string, handler Handler) error

	// QueueSubscribe permite load balancing entre instâncias do mesmo módulo.
	QueueSubscribe(topic, queue string, handler Handler) error
}
