package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	*echo.Echo
}

func New() *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// e.Debug = false
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	s := &Server{e}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
}
