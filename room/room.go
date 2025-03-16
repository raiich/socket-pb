package room

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/task"
	"github.com/raiich/socket-pb/stream"
)

type ID string

var IDType = reflect.TypeOf([0]ID{}).Elem()

type Room struct {
	ID         ID
	Dispatcher Dispatcher
	Parent     *Manager
	Clients    map[ClientID]Client
	Handler    Handler
	currentID  ClientID
}

func (r *Room) NextClientID() ClientID {
	r.currentID++
	return r.currentID
}

func (r *Room) AddClient(client Client) {
	r.Clients[client.ID()] = client
}

func (r *Room) Kick(ctx context.Context, cid ClientID) error {
	c, ok := r.Clients[cid]
	if !ok {
		return fmt.Errorf("client not found: %v", cid)
	}
	delete(r.Clients, cid)
	c.StartShutdown(ctx)
	return nil
}

type Handler interface {
	OnPayload(clientID ClientID, payload *stream.Payload)
}

type Manager struct {
	settings *ManagerSettings
	mu       sync.Mutex
	roomMap  map[ID]*Room
}

func (m *Manager) GetOrCreate(id ID) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	room, ok := m.roomMap[id]
	if !ok {
		dispatcher := m.settings.Dispatcher()
		room = &Room{
			ID:         id,
			Dispatcher: dispatcher,
			Parent:     m,
			Clients:    map[ClientID]Client{},
		}
		m.roomMap[id] = room
		go func() {
			log.OnError(dispatcher.Launch())
		}()
	}
	return room
}

func (m *Manager) Delete(id ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room, ok := m.roomMap[id]
	if !ok {
		return
	}
	log.OnError(room.Dispatcher.Stop())
	delete(m.roomMap, id)
}

func NewManager(settings *ManagerSettings) *Manager {
	return &Manager{
		settings: settings,
		roomMap:  make(map[ID]*Room),
	}
}

type ManagerSettings struct {
	Context        context.Context
	DispatcherType DispatcherType
}

func (s *ManagerSettings) Dispatcher() Dispatcher {
	ctx := s.Context
	switch s.DispatcherType {
	case DispatcherTypeAsync:
		return task.NewAsyncDispatcher(ctx)
	case DispatcherTypeMutex:
		return task.NewMutexDispatcher(ctx)
	default:
		return task.NewAsyncDispatcher(ctx)
	}
}

type App interface {
	OnClientJoin(client Client)
}
