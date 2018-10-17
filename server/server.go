package server

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net"
	"time"
)

type Server struct {
	*echo.Echo
	config       Config
	HttpListener net.Listener
}

// Ripped straight from https://golang.org/src/net/http/server.go. Have to
// define our own Listener because we need to be able to Close() it manually.
// (not really sure why it's not exposed in the http package)
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = NewDefaultClientConfig()
	}

	ln, err := newListener(cfg.IncomingPort)
	if err != nil {
		return nil, err
	}

	return &Server{
		Echo:         echo.New(),
		HttpListener: ln,
	}, nil
}

func NewDefaultClientConfig() *Config {
	return &Config{IncomingPort: "8080"}
}

func (s *Server) Run(route func(*echo.Echo)) error {
	s.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowMethods: []string{"GET", "PUT", "UPDATE", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			echo.HeaderAuthorization,
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAccessControlRequestHeaders,
		},
	}))

	route(s.Echo)

	return s.Server.Serve(s.HttpListener)
}

func (s *Server) Config() Config {
	return s.config
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func newListener(port string) (net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, err
	}

	ln = tcpKeepAliveListener{ln.(*net.TCPListener)}

	return ln, nil
}
