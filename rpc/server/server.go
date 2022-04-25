package server

import (
	//"errors"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"time"

	"github.com/SugierPawel/player/rpc/core"
)

type Server struct {
	Port     int
	Sleep    time.Duration
	listener net.Listener
}

func (s *Server) Close() (err error) {
	if s.listener != nil {
		err = s.listener.Close()
	}
	return
}

func (s *Server) Start() (err error) {

	log.Printf(">> ListenUDP s.Port: %d", s.Port)
	log.Println()

	rpc.Register(&core.Handler{
		Sleep: s.Sleep,
	})

	s.listener, err = net.Listen("tcp", ":"+strconv.Itoa(int(s.Port)))
	if err != nil {
		log.Panicln(err)
		return
	}
	rpc.Accept(s.listener)

	return
}
