### v3.0.0
* fully reimplemented with API change (see Migrated to V3 in README.MD)
### v2.1.3
- client: fully process workers channel before closing connection
- fix closing server connection
- use atomic for connection context
- server rooms: add api to retrieve all connections
### v2.1.2
- add worker queue for client to prevent read blocking
### v2.1.1
- fix bug in server.BroadcastToAll method
### v2.1.0
- migrate to go modules
- add ping method
### v2.0.2
- fixed deadlock on ack data chan when Acker.Notify() executed before Acker.Await()
### v2.0.1
- fixed data race when pushing response data to EmitWithAck channel
- add using buffer from pool in parser.EncodeEvent to decrease allocations
### v2.0.0
**NOTE**: protocol has changed, release is not compatible with 1.x.x
- add `OnWithAck` `EmitWithAck` methods for sync RPC
- add `sync.Pool` using
- add tests
