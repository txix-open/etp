package etp

import "sync"

const (
	// idsRoom is the internal room name for tracking all connections.
	idsRoom = "__id"
)

// Rooms manages groups of connections for broadcasting.
// It is safe for concurrent use.
type Rooms struct {
	mu    sync.RWMutex
	rooms map[string]map[string]*Conn
}

// newRooms creates a new Rooms instance.
func newRooms() *Rooms {
	return &Rooms{
		rooms: make(map[string]map[string]*Conn),
	}
}

// Get retrieves a connection by its ID from the internal room.
// It returns the connection and a boolean indicating whether it exists.
func (s *Rooms) Get(connId string) (*Conn, bool) {
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

// Join adds a connection to the specified rooms.
// New rooms are created if they do not exist.
func (s *Rooms) Join(conn *Conn, rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		if conns, ok := s.rooms[room]; ok {
			conns[conn.Id()] = conn
		} else {
			s.rooms[room] = map[string]*Conn{
				conn.Id(): conn,
			}
		}
	}
	s.mu.Unlock()
}

// LeaveByConnId removes a connection from the specified rooms.
// Empty rooms are automatically deleted.
func (s *Rooms) LeaveByConnId(id string, rooms ...string) {
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

// Len returns the number of connections in the specified room.
func (s *Rooms) Len(room string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.rooms[room])
}

// Clear removes the specified rooms.
func (s *Rooms) Clear(rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		delete(s.rooms, room)
	}
	s.mu.Unlock()
}

// Rooms returns a slice of all room names, excluding the internal room.
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

// ToBroadcast returns all connections in the specified rooms for broadcasting.
// Duplicate connections are included multiple times if they belong to multiple rooms.
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

// AllConns returns all connections managed by the server.
func (s *Rooms) AllConns() []*Conn {
	return s.ToBroadcast(idsRoom)
}

// add adds a connection to the internal room.
func (s *Rooms) add(conn *Conn) {
	s.Join(conn, idsRoom)
}

// remove removes a connection from all rooms.
func (s *Rooms) remove(conn *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for room, conns := range s.rooms {
		delete(conns, conn.Id())
		if len(conns) == 0 {
			delete(s.rooms, room)
		}
	}
}
