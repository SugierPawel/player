package core

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"time"
)

var RPC_PORT int = 1337

type CMDResponse struct {
	Message string
	Ok      bool
}

const (
	AddRTPsourceRequest string = "AddRTPsource"
	DelRTPsourceRequest string = "DelRTPsource"
)

type StreamConfig struct {
	StreamName  string
	Request     string
	IPIn        string
	PortIn      int
	ChannelName string
}

const HandlerName = "Handler.Execute"

type Handler struct {
	Sleep time.Duration
}

var StreamConfigChan = make(chan *StreamConfig, 1)

func (h *Handler) Execute(sc *StreamConfig, res *CMDResponse) (err error) {
	sc.StreamName = sc.IPIn + ":" + strconv.Itoa(sc.PortIn)
	StreamConfigChan <- sc
	if h.Sleep != 0 {
		time.Sleep(h.Sleep)
	}
	res.Ok = true
	res.Message = "OK:" + sc.Request
	return
}

type CMDServer struct {
	Port     int
	Sleep    time.Duration
	listener net.Listener
}

func (s *CMDServer) Close() (err error) {
	if s.listener != nil {
		err = s.listener.Close()
	}
	return
}

func (s *CMDServer) Start() (err error) {
	log.Printf("Serwer nasłuchuje komend na porcie: %d", s.Port)
	rpc.Register(&Handler{
		Sleep: s.Sleep,
	})
	s.listener, err = net.Listen("tcp", ":"+strconv.Itoa(int(s.Port)))
	if err != nil {
		return
	}
	rpc.Accept(s.listener)
	return
}

func StartCMDServer() {
	server := &CMDServer{
		Sleep: 0,
		Port:  RPC_PORT,
	}
	defer server.Close()
	go server.Start()
}
func tsToTime(ts uint32) string {
	d := time.Unix(int64(ts), 0)
	h, m, s := d.Clock()
	return fmt.Sprintf("%d:%d:%d", h, m, s)
}
