package etp

import "sync"

const (
	idsRoom = "__id"
)

type Rooms struct {
	mu    sync.RWMutex
	rooms map[string]map[uint64]*Conn
}

func newRooms() *Rooms {
	return &Rooms{
		rooms: make(map[string]map[uint64]*Conn),
	}
}

func (s *Rooms) Add(conn *Conn) {
	s.Join(conn, idsRoom)
}

func (s *Rooms) Remove(conn *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for room, conns := range s.rooms {
		delete(conns, conn.Id())
		if len(conns) == 0 {
			delete(s.rooms, room)
		}
	}
}

func (s *Rooms) Get(connId uint64) (*Conn, bool) {
	s.mu.RLock()
	var (
		conn *Conn
		ok   bool
	)
	idRoom, roomExist := s.rooms[idsRoom]
	if roomExist {
		conn, ok = idRoom[connId]
	}
	s.mu.RUnlock()
	return conn, ok
}

func (s *Rooms) Join(conn *Conn, rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		if conns, ok := s.rooms[room]; ok {
			conns[conn.Id()] = conn
		} else {
			s.rooms[room] = map[uint64]*Conn{
				conn.Id(): conn,
			}
		}
	}
	s.mu.Unlock()
}

func (s *Rooms) LeaveByConnId(id uint64, rooms ...string) {
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

func (s *Rooms) Len(room string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.rooms[room])
}

func (s *Rooms) Clear(rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		delete(s.rooms, room)
	}
	s.mu.Unlock()
}

func (s *Rooms) Rooms() []string {
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

func (s *Rooms) ToBroadcast(rooms ...string) []*Conn {
	s.mu.RLock()
	result := make([]*Conn, 0)
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

func (s *Rooms) AllConns() []*Conn {
	return s.ToBroadcast(idsRoom)
}
