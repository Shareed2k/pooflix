package server

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net"
	"time"
)

type Server struct {
	echo         *echo.Echo
	httpListener net.Listener
}

func New() (*Server, error) {
	ln, err := newListener("8007")
	if err != nil {
		return nil, err
	}

	return &Server{httpListener: ln}, nil
}

func (s *Server) Run(route func(*echo.Echo)) error {
	s.echo = echo.New()
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowMethods: []string{"GET", "PUT", "UPDATE", "POST", "DELETE"},
		AllowHeaders: []string{
			echo.HeaderAuthorization,
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAccessControlRequestHeaders,
		},
	}))

	route(s.echo)

	return s.echo.Server.Serve(s.httpListener)
}

// Ripped straight from https://golang.org/src/net/http/server.go. Have to
// define our own Listener because we need to be able to Close() it manually.
// (not really sure why it's not exposed in the http package)
type tcpKeepAliveListener struct {
	*net.TCPListener
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
