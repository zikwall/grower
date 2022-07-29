package syslog

import (
	"context"
	"sync"

	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"

	"github.com/zikwall/grower/pkg/log"
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
	defer func() {
		close(s.channel)
		log.Info("syslog awaiter successfully finished")
	}()
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
	log.Info("syslog server is ready to receive messages...")
	s.server.Wait()
	return nil
}

// Drop method implements drop.Drop interface
// Drop method cleans up all resources, closes channels and waits for completion of all goroutines
func (s *Server) Drop() error {
	// first, stop syslog daemon to avoid cases when writing to a closed channel will be performed
	err := s.server.Kill()
	// finally, waiting for the completion of all goroutines
	s.wg.Wait()
	// return an error, if any
	return err
}

// DropMsg method implements drop.Debug interface
// DropMsg writes to log fact that Syslog was successfully destroyed
func (s *Server) DropMsg() string {
	return "syslog server was successfully destroyed"
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
