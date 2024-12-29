package quicx

import (
	"context"
	"crypto/tls"
	"net"
	"runtime/debug"

	"github.com/quic-go/quic-go"
)

type Handler interface {
	Handle(context.Context, net.Conn)
}

type listener struct {
	conn       *net.UDPConn
	quicServer *quic.Listener
}

var _ net.Listener = &listener{}

// Listen creates a QUIC listener on the given network interface
func Listen(network, laddr string, tlsConfig *tls.Config, quicConfig *quic.Config) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr(network, laddr)
	if err != nil {
		return nil, &net.OpError{Op: "listen", Net: network, Source: nil, Addr: nil, Err: err}
	}
	conn, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, err
	}
	ln, err := quic.Listen(conn, tlsConfig, quicConfig)
	if err != nil {
		return nil, err
	}
	return &listener{
		conn:       conn,
		quicServer: ln,
	}, nil
}

// Accept waits for and returns the next connection to the listener.
func (x *listener) Accept() (net.Conn, error) {
	conn, err := x.quicServer.Accept(context.Background())
	if err != nil {
		return nil, err
	}
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		return nil, err
	}
	qconn := &Quic{
		conn:   x.conn,
		qconn:  conn,
		stream: stream,
	}
	return qconn, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (x *listener) Close() error {
	return x.quicServer.Close()
}

// Addr returns the listener's network address.
func (x *listener) Addr() net.Addr {
	return x.quicServer.Addr()
}

// ListenAndServe binds port and handle requests, blocking until close
func ListenAndServe(ctx context.Context, listener net.Listener, handler Handler) {
	defer func() {
		if e := recover(); e != nil {
			debug.PrintStack()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		handler.Handle(ctx, conn)
	}
}