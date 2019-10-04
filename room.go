package etp

import "sync"

type RoomStore interface {
	Add(Conn)
	Remove(Conn)
	Get(id string) (Conn, bool)
	Join(conn Conn, rooms ...string)
	Leave(conn Conn, rooms ...string)
	LeaveByConnId(id string, rooms ...string)
	Len(room string) int
	Clear(rooms ...string)
	Rooms() []string
	ToBroadcast(rooms ...string) []Conn
}

const (
	idsRoom = "__id"
)

type roomStore struct {
	mu    sync.RWMutex
	rooms map[string]map[string]Conn
}

func (s *roomStore) Add(conn Conn) {
	s.Join(conn, idsRoom)
}

func (s *roomStore) Remove(conn Conn) {
	s.LeaveByConnId(conn.ID(), idsRoom)
}

func (s *roomStore) Get(connId string) (Conn, bool) {
	s.mu.RLock()
	var (
		conn Conn
		ok   bool
	)
	idRoom, roomExist := s.rooms[idsRoom]
	if roomExist {
		conn, ok = idRoom[connId]
	}
	s.mu.RUnlock()
	return conn, ok
}

func (s *roomStore) Join(conn Conn, rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		if conns, ok := s.rooms[room]; ok {
			conns[conn.ID()] = conn
		} else {
			s.rooms[room] = map[string]Conn{conn.ID(): conn}
		}
	}
	s.mu.Unlock()
}

func (s *roomStore) Leave(conn Conn, rooms ...string) {
	s.LeaveByConnId(conn.ID(), rooms...)
}

func (s *roomStore) LeaveByConnId(id string, rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		if conns, ok := s.rooms[room]; ok {
			delete(conns, id)
			if len(conns) == 0 {
				delete(s.rooms, room)
			}
		}
	}
	s.mu.Unlock()
}

func (s *roomStore) Len(room string) int {
	s.mu.RLock()
	var length int
	if conns, ok := s.rooms[room]; ok {
		length = len(conns)
	}
	s.mu.RUnlock()
	return length
}

func (s *roomStore) Clear(rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		delete(s.rooms, room)
	}
	s.mu.Unlock()
}

func (s *roomStore) Rooms() []string {
	s.mu.RLock()
	result := make([]string, 0, len(s.rooms))
	for room := range s.rooms {
		if room != idsRoom {
			result = append(result, room)
		}
	}
	s.mu.RUnlock()
	return result
}

func (s *roomStore) ToBroadcast(rooms ...string) []Conn {
	s.mu.RLock()
	result := make([]Conn, 0)
	for _, room := range rooms {
		if conns, ok := s.rooms[room]; ok {
			for _, conn := range conns {
				result = append(result, conn)
			}
		}
	}
	s.mu.RUnlock()
	return result
}

func NewRoomStore() RoomStore {
	return &roomStore{
		rooms: make(map[string]map[string]Conn),
	}
}
