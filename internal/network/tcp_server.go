package network

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Novip1906/my-redis/internal/aof"
	"github.com/Novip1906/my-redis/internal/compute"
)

type TCPServer struct {
	wg       sync.WaitGroup
	port     string
	parser   *compute.Parser
	aof      *aof.AOF
	log      *slog.Logger
	listener net.Listener
	conns    map[net.Conn]struct{}
	mu       sync.Mutex
}

func NewTCPServer(port string, parser *compute.Parser, aof *aof.AOF, log *slog.Logger) *TCPServer {
	return &TCPServer{
		port:   port,
		parser: parser,
		aof:    aof,
		log:    log,
		conns:  make(map[net.Conn]struct{}),
	}
}

func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.port)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	s.listener = listener

	s.log.Info("TCP Server started", "port", s.port)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				s.log.Info("Server stopped accepting new connections")
				return nil
			}
			s.log.Error("Connection error", "error", err)
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) Stop() {
	s.listener.Close()

	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	log := s.log.With("client", remoteAddr)
	log.Info("New connection")

	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		conn.Close()
		s.wg.Done()
	}()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		commandLine := scanner.Text()

		response, saveToAOF := s.parser.ProcessCommand(commandLine)

		response += "\n"

		if strings.HasPrefix(strings.ToUpper(commandLine), "QUIT") {
			conn.Write([]byte(response))
			break
		}

		if saveToAOF {
			if err := s.aof.Write(commandLine); err != nil {
				s.log.Error("Failed to write to AOF", "error", err)
			}
		}

		conn.Write([]byte(response))
	}

	log.Info("Connection closed")
}
