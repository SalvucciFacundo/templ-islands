package main

import "sync"

// Broker difunde eventos a los suscriptores SSE conectados. Es el "hub" de la
// app (la libreria da los helpers; el hub es responsabilidad del proyecto).
type Broker struct {
	mu      sync.Mutex
	clients map[chan map[string]any]struct{}
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[chan map[string]any]struct{})}
}

// Subscribe registra un canal para un nuevo cliente SSE.
func (b *Broker) Subscribe() chan map[string]any {
	ch := make(chan map[string]any, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe cierra el canal de un cliente que se desconecto.
func (b *Broker) Unsubscribe(ch chan map[string]any) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

// Publish envia msg a todos los clientes; si un cliente esta lento, se dropea.
func (b *Broker) Publish(msg map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
