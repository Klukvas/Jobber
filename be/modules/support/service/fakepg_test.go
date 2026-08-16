package service

import (
	"net"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgproto3"
)

// fakePG is a minimal in-process PostgreSQL wire-protocol server used to make
// the concrete *users.UserRepository (which requires a real *pgxpool.Pool)
// return a deterministic row without any external database or Docker.
//
// It speaks just enough of the protocol for pgx's QueryRow: the startup
// handshake plus the extended-query flow (Parse/Bind/Describe/Execute/Sync).
// It answers every query with the single canned row supplied to newFakePG,
// or with an error response when errText is set. It is NOT a database — it is a
// protocol mock, in the same spirit as an httptest.Server for HTTP clients.
type pgCol struct {
	name string
	oid  uint32
}

type fakePG struct {
	ln      net.Listener
	cols    []pgCol  // column names + type OIDs for the row description
	row     [][]byte // one row of text-encoded values (nil => empty result)
	errText string   // if set, respond to every query with this error
	wg      sync.WaitGroup
	mu      sync.Mutex
	closed  bool
	conns   map[net.Conn]struct{} // live accepted connections
}

// newFakePG starts a fake server on a loopback port and returns it. The caller
// must Close it when done. cols/row describe the row returned for any query.
func newFakePG(t *testing.T, cols []pgCol, row [][]byte, errText string) *fakePG {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fakePG: listen: %v", err)
	}
	f := &fakePG{ln: ln, cols: cols, row: row, errText: errText, conns: make(map[net.Conn]struct{})}
	f.wg.Add(1)
	go f.acceptLoop()
	return f
}

// addr returns the loopback host:port the server is listening on.
func (f *fakePG) addr() string { return f.ln.Addr().String() }

// dsn returns a libpq DSN pointing at the fake server.
func (f *fakePG) dsn() string {
	return "postgres://u:p@" + f.addr() + "/db?sslmode=disable"
}

func (f *fakePG) Close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	// Force-close every accepted connection so serve goroutines blocked in
	// Receive() unblock immediately — independent of when the pgx pool closes.
	for c := range f.conns {
		_ = c.Close()
	}
	f.mu.Unlock()
	_ = f.ln.Close()
	f.wg.Wait()
}

func (f *fakePG) acceptLoop() {
	defer f.wg.Done()
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return // listener closed
		}
		f.mu.Lock()
		if f.closed {
			f.mu.Unlock()
			_ = conn.Close()
			continue
		}
		f.conns[conn] = struct{}{}
		f.mu.Unlock()

		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			defer func() {
				f.mu.Lock()
				delete(f.conns, conn)
				f.mu.Unlock()
				_ = conn.Close()
			}()
			_ = f.serve(conn)
		}()
	}
}

func (f *fakePG) serve(conn net.Conn) error {
	be := pgproto3.NewBackend(conn, conn)

	if err := f.handleStartup(be, conn); err != nil {
		return err
	}

	for {
		msg, err := be.Receive()
		if err != nil {
			return err
		}
		switch m := msg.(type) {
		case *pgproto3.Query:
			// Simple query protocol (unused by QueryRow-with-args but handled
			// for completeness).
			if err := f.writeResult(conn, true, true); err != nil {
				return err
			}
		case *pgproto3.Parse:
			if err := writeMsgs(conn, &pgproto3.ParseComplete{}); err != nil {
				return err
			}
		case *pgproto3.Bind:
			if err := writeMsgs(conn, &pgproto3.BindComplete{}); err != nil {
				return err
			}
		case *pgproto3.Describe:
			// 'S' (statement) describe is part of the prepare round: reply with
			// the parameter description followed by the row description. 'P'
			// (portal) describe just needs the row description.
			if m.ObjectType == 'S' {
				if err := writeMsgs(conn, &pgproto3.ParameterDescription{ParameterOIDs: []uint32{25}}); err != nil {
					return err
				}
			}
			if err := writeMsgs(conn, f.rowDescription()); err != nil {
				return err
			}
		case *pgproto3.Execute:
			// The row description was already emitted in response to Describe;
			// here we only send the data row + command completion.
			if err := f.writeResult(conn, false, false); err != nil {
				return err
			}
		case *pgproto3.Sync:
			if err := writeMsgs(conn, &pgproto3.ReadyForQuery{TxStatus: 'I'}); err != nil {
				return err
			}
		case *pgproto3.Terminate:
			return nil
		default:
			// Ignore anything else.
		}
	}
}

// rowDescription builds the RowDescription for the canned columns, honoring
// each column's declared type OID so pgx scans into the right Go type.
func (f *fakePG) rowDescription() *pgproto3.RowDescription {
	fields := make([]pgproto3.FieldDescription, len(f.cols))
	for i, c := range f.cols {
		fields[i] = pgproto3.FieldDescription{
			Name:         []byte(c.name),
			DataTypeOID:  c.oid,
			DataTypeSize: -1,
			TypeModifier: -1,
			Format:       0,
		}
	}
	return &pgproto3.RowDescription{Fields: fields}
}

// writeResult emits either the canned row (or empty result) or an error
// response. When withDesc is true the row description precedes the data; when
// withRFQ is true a ReadyForQuery is appended (used by the simple-query path).
func (f *fakePG) writeResult(conn net.Conn, withDesc, withRFQ bool) error {
	if f.errText != "" {
		msgs := []pgproto3.Message{
			&pgproto3.ErrorResponse{Severity: "ERROR", Code: "XX000", Message: f.errText},
		}
		if withRFQ {
			msgs = append(msgs, &pgproto3.ReadyForQuery{TxStatus: 'I'})
		}
		return writeMsgs(conn, msgs...)
	}

	var msgs []pgproto3.Message
	if withDesc {
		msgs = append(msgs, f.rowDescription())
	}
	if f.row != nil {
		msgs = append(msgs, &pgproto3.DataRow{Values: f.row})
		msgs = append(msgs, &pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
	} else {
		msgs = append(msgs, &pgproto3.CommandComplete{CommandTag: []byte("SELECT 0")})
	}
	if withRFQ {
		msgs = append(msgs, &pgproto3.ReadyForQuery{TxStatus: 'I'})
	}
	return writeMsgs(conn, msgs...)
}

func (f *fakePG) handleStartup(be *pgproto3.Backend, conn net.Conn) error {
	for {
		startup, err := be.ReceiveStartupMessage()
		if err != nil {
			return err
		}
		switch startup.(type) {
		case *pgproto3.StartupMessage:
			return writeMsgs(conn,
				&pgproto3.AuthenticationOk{},
				&pgproto3.ParameterStatus{Name: "server_version", Value: "15.0"},
				&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"},
				&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: 1},
				&pgproto3.ReadyForQuery{TxStatus: 'I'},
			)
		case *pgproto3.SSLRequest:
			// Deny SSL and loop to read the real startup message.
			if _, err := conn.Write([]byte("N")); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func writeMsgs(conn net.Conn, msgs ...pgproto3.Message) error {
	var buf []byte
	var err error
	for _, m := range msgs {
		buf, err = m.Encode(buf)
		if err != nil {
			return err
		}
	}
	_, err = conn.Write(buf)
	return err
}
