package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsBus struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func NewNatsBus(url, env string) (*NatsBus, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("erro ao criar contexto JetStream: %w", err)
	}

	// [PHASE 2] Reliability Configuration
	replicas := 1
	maxAge := 24 * time.Hour // Deletion after 1 day for dev

	if env == "production" {
		replicas = 3
		maxAge = 7 * 24 * time.Hour // Deletion after 7 days for prod
	}

	// [DEV-FIX] Garantir que existe uma Stream "EVENTS" padrão
	// Inicialmente apenas com "events.>", outros tópicos serão adicionados dinamicamente no Subscribe
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"events.>"},
		Storage:  jetstream.FileStorage,
		Replicas: replicas,
		MaxAge:   maxAge,
	})
	if err != nil {
		// Apenas loga erro (fmt) mas não aborta, pois pode ser conflito que se resolve depois
		fmt.Printf("⚠️ Aviso: Falha ao criar stream EVENTS padrão: %v\n", err)
	}

	return &NatsBus{
		nc: nc,
		js: js,
	}, nil
}

func (b *NatsBus) Close() {
	if b.nc != nil {
		b.nc.Close()
	}
}

// Publish envia um evento para o ecossistema.
func (b *NatsBus) Publish(topic string, payload interface{}) error {
	var data []byte
	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal payload type %T: %w", payload, err)
		}
		data = encoded
	}

	ctx := context.Background()

	// Verificar se o subject está em alguma stream existente
	_, err := b.js.StreamNameBySubject(ctx, topic)
	if err != nil {
		// Subject não está em nenhuma stream, precisamos adicionar à stream EVENTS
		streamName := "EVENTS"
		s, err := b.js.Stream(ctx, streamName)
		if err != nil {
			// Stream EVENTS não existe, criar com o subject
			_, err = b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
				Name:     streamName,
				Subjects: []string{"events.>", topic},
				Storage:  jetstream.FileStorage,
			})
			if err != nil {
				return fmt.Errorf("erro ao criar stream EVENTS para %s: %w", topic, err)
			}
		} else {
			// Stream existe, verificar se precisa adicionar o subject
			cfg := s.CachedInfo().Config
			hasSubject := false
			for _, sub := range cfg.Subjects {
				// Verificar se o subject corresponde a algum padrão existente
				if sub == topic || matchesSubjectPattern(sub, topic) {
					hasSubject = true
					break
				}
			}

			if !hasSubject {
				// Adicionar o subject à lista
				cfg.Subjects = append(cfg.Subjects, topic)
				_, err = b.js.UpdateStream(ctx, cfg)
				if err != nil {
					return fmt.Errorf("erro ao atualizar stream EVENTS com %s: %w", topic, err)
				}
			}
		}
	}

	// Publicar o evento
	_, err = b.js.Publish(ctx, topic, data)
	return err
}

// matchesSubjectPattern verifica se um subject corresponde a um padrão (ex: "events.>" corresponde a "events.qualquer.coisa")
func matchesSubjectPattern(pattern, subject string) bool {
	if pattern == subject {
		return true
	}
	// Verificar padrões com wildcards
	if strings.HasSuffix(pattern, ">") {
		prefix := strings.TrimSuffix(pattern, ">")
		return strings.HasPrefix(subject, prefix)
	}
	return false
}

// Subscribe escuta eventos de um tópico.
func (b *NatsBus) Subscribe(topic string, handler Handler) error {
	ctx := context.Background()

	// 1. Tenta descobrir se já existe uma stream para este tópico
	streamName, err := b.js.StreamNameBySubject(ctx, topic)
	if err != nil {
		// 2. Se não existe, vamos adicionar este tópico à stream "EVENTS"
		streamName = "EVENTS"

		// Obtém info da stream EVENTS
		s, err := b.js.Stream(ctx, streamName)
		if err != nil {
			// Se nem a EVENTS existe, tenta criar agora (Fallback extremo)
			_, err = b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
				Name:     streamName,
				Subjects: []string{"events.>", topic},
				Storage:  jetstream.FileStorage,
			})
			if err != nil {
				return fmt.Errorf("erro ao criar stream EVENTS para %s: %w", topic, err)
			}
		} else {
			// Se existe, atualiza os subjects
			cfg := s.CachedInfo().Config
			// Verifica se já tem o subject (pra não duplicar)
			hasSubject := false
			for _, sub := range cfg.Subjects {
				if sub == topic {
					hasSubject = true
					break
				}
			}

			if !hasSubject {
				cfg.Subjects = append(cfg.Subjects, topic)
				_, err = b.js.UpdateStream(ctx, cfg)
				if err != nil {
					return fmt.Errorf("erro ao atualizar stream EVENTS com %s: %w", topic, err)
				}
			}
		}
	}

	// 3. Cria consumer efêmero
	consumer, err := b.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		FilterSubject: topic,
	})

	if err != nil {
		return fmt.Errorf("erro ao criar consumer para %s na stream %s: %w", topic, streamName, err)
	}

	_, err = consumer.Consume(func(msg jetstream.Msg) {
		wrapper := &NatsMessage{msg: msg}
		if err := handler(wrapper); err != nil {
			msg.Nak()
		} else {
			msg.Ack()
		}
	})

	return err
}

// QueueSubscribe permite load balancing.
func (b *NatsBus) QueueSubscribe(topic, queue string, handler Handler) error {
	ctx := context.Background()

	streamName, err := b.js.StreamNameBySubject(ctx, topic)
	if err != nil {
		streamName = "EVENTS"
	}

	// Queue Group no JetStream = Durable Consumer compartilhado
	consumer, err := b.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          queue, // Durable Name = Queue Group
		Durable:       queue,
		FilterSubject: topic,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("erro ao criar queue consumer %s para %s: %w", queue, topic, err)
	}

	_, err = consumer.Consume(func(msg jetstream.Msg) {
		wrapper := &NatsMessage{msg: msg}
		if err := handler(wrapper); err != nil {
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
	return err
}

// --- Message Wrapper ---

type NatsMessage struct {
	msg jetstream.Msg
}

func (m *NatsMessage) ID() string {
	meta, _ := m.msg.Metadata()
	if meta != nil {
		return fmt.Sprintf("%d", meta.Sequence.Stream)
	}
	return "unknown"
}

func (m *NatsMessage) Topic() string {
	return m.msg.Subject()
}

func (m *NatsMessage) Payload() []byte {
	return m.msg.Data()
}

func (m *NatsMessage) Ack() error {
	return m.msg.Ack()
}

func (m *NatsMessage) Nak() error {
	return m.msg.Nak()
}
