package syslog

import (
	"context"
	"sync"

	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"

	"github.com/zikwall/ck-nginx/pkg/log"
)

const (
	ListenerTCP = "tcp"
	ListenerUPD = "upd"
	ListenerUDS = "unix"
)

type Handler func(format.LogParts)

type Server struct {
	cfg     *Cfg
	handler Handler
	server  *syslog.Server
	channel syslog.LogPartsChannel
	wg      *sync.WaitGroup
}

func (s *Server) SetHandler(handler Handler) {
	s.handler = handler
}

func (s *Server) Await(ctx context.Context) error {
	for _, listener := range s.cfg.Listeners {
		switch listener {
		case ListenerTCP:
			if err := s.server.ListenTCP(s.cfg.TCP); err != nil {
				return err
			}
			log.Infof("listen TCP on: %s", s.cfg.TCP)
		case ListenerUPD:
			if err := s.server.ListenUDP(s.cfg.UPD); err != nil {
				return err
			}
			log.Infof("listen UDP on: %s", s.cfg.UPD)
		case ListenerUDS:
			if err := s.server.ListenUnixgram(s.cfg.Unix); err != nil {
				return err
			}
			log.Infof("listen UNIX socket on: %s", s.cfg.Unix)
		}
	}
	for i := 1; i <= s.cfg.Parallelism; i++ {
		s.wg.Add(1)
		go func(n int) {
			log.Infof("run syslog channel listener %d", n)
			defer func() {
				s.wg.Done()
				log.Infof("stop syslog channel listener %d", n)
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case logParts := <-s.channel:
					s.handler(logParts)
				}
			}
		}(i)
	}
	if err := s.server.Boot(); err != nil {
		return err
	}
	log.Info("SYSLOG SERVER RUN")
	s.server.Wait()
	log.Info("SYSLOG SERVER STOP")
	return nil
}

func (s *Server) Drop() error {
	close(s.channel)
	s.wg.Wait()
	return s.server.Kill()
}

func (s *Server) DropMsg() string {
	return "kill syslog server"
}

func NewServer(cfg *Cfg) *Server {
	s := &Server{
		cfg: cfg,
		wg:  &sync.WaitGroup{},
	}
	s.channel = make(syslog.LogPartsChannel, cfg.BufSize+1)
	handler := syslog.NewChannelHandler(s.channel)
	s.server = syslog.NewServer()
	s.server.SetFormat(syslog.RFC3164)
	s.server.SetHandler(handler)
	return s
}
