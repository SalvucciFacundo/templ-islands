package main

import "sync"

// BrokerEvent es un evento con id secuencial, para resiliencia SSE
// (Last-Event-ID permite reanudar desde el ultimo evento recibido).
type BrokerEvent struct {
	ID   int
	Data map[string]any
}

// Broker difunde eventos a los suscriptores SSE conectados y guarda un
// historial acotado para permitir la reanudacion por Last-Event-ID. Es el
// "hub" de la app (la libreria da los helpers; el hub es responsabilidad
// del proyecto).
type Broker struct {
	mu      sync.Mutex
	clients map[chan BrokerEvent]struct{}
	events  []BrokerEvent
	nextID  int
	maxHist int
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[chan BrokerEvent]struct{}), maxHist: 200}
}

// Subscribe registra un canal para un nuevo cliente SSE.
func (b *Broker) Subscribe() chan BrokerEvent {
	ch := make(chan BrokerEvent, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe cierra el canal de un cliente que se desconecto.
func (b *Broker) Unsubscribe(ch chan BrokerEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

// Publish asigna un id, guarda el evento en el historial y lo difunde.
func (b *Broker) Publish(msg map[string]any) BrokerEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	ev := BrokerEvent{ID: b.nextID, Data: msg}
	b.events = append(b.events, ev)
	if len(b.events) > b.maxHist {
		b.events = b.events[len(b.events)-b.maxHist:]
	}
	for ch := range b.clients {
		select {
		case ch <- ev:
		default: // cliente lento: dropear (el historial cubre la perdida)
		}
	}
	return ev
}

// History devuelve los eventos con ID > after, para reanudar por Last-Event-ID.
func (b *Broker) History(after int) []BrokerEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := []BrokerEvent{}
	for _, ev := range b.events {
		if ev.ID > after {
			out = append(out, ev)
		}
	}
	return out
}
