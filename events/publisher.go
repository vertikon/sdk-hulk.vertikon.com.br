package events

// Publisher fornece métodos auxiliares para publicação de eventos tipados.
type Publisher struct {
	bus Bus
}

// NewPublisher cria um novo Publisher.
func NewPublisher(bus Bus) *Publisher {
	return &Publisher{bus: bus}
}

// PublishJSON publica um evento serializando o payload como JSON.
func (p *Publisher) PublishJSON(topic string, payload interface{}) error {
	return p.bus.Publish(topic, payload)
}

// PublishBytes publica um evento com payload em bytes.
func (p *Publisher) PublishBytes(topic string, payload []byte) error {
	return p.bus.Publish(topic, payload)
}
