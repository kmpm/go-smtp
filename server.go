// A ESMTP server library.
package smtp

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
)

// A SMTP server.
type Server struct {
	// The server backend.
	Backend           Backend
	// The server configuration.
	Config            *Config
	// The server TLS configuration.
	TLSConfig         *tls.Config

	listener          net.Listener
	caps              []string
}

// Create a new SMTP server.
func New(l net.Listener, cfg *Config, bkd Backend) *Server {
	return &Server{
		Backend:  bkd,
		Config:   cfg,
		listener: l,
		caps:     []string{"PIPELINING", "8BITMIME"},
	}
}

// Listen for incoming connections.
func (s *Server) Listen() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(&Conn{
			server: s,
			conn:   conn,
			reader: bufio.NewReader(conn),
		})
	}
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) handleConn(c *Conn) error {
	defer c.Close()
	c.greet()

	for {
		line, err := c.readLine()
		if err == nil {
			cmd, arg, err := parseCmd(line)
			if err != nil {
				c.nbrErrors++
				c.Write("501", "Bad command")
				continue
			}

			c.handle(cmd, arg)
		} else {
			if err == io.EOF {
				return nil
			}

			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				c.Write("221", "Idle timeout, bye bye")
				return nil
			}

			c.Write("221", "Connection error, sorry")
			return err
		}
	}
}
