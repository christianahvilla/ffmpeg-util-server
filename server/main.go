package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/christianahvilla/ffmpegutil"
	//"ffmpegutil"

	"github.com/takama/daemon"
)

const (
	// name of the service
	name        = "ffmpeg-video"
	description = "comprres videos"

	// port which daemon should be listen
	port = ":9977"
)

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

func main() {
	srv, err := daemon.New(name, description)
	if err != nil {
		ffmpegutil.WriteLog(ffmpegutil.Error, "\nError: "+err.Error())
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()

	if err != nil {
		ffmpegutil.WriteLog(status, "\nError: "+err.Error())
		os.Exit(1)
	}

	ffmpegutil.WriteLog(ffmpegutil.Info, status)

	//ffmpegutil.Server()
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	usage := "Usage: main install | remove | start | stop | status"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	// Do something, call your goroutines, etc

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Set up listener for defined host and port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return "Possibly was a problem with the port binding", err
	}

	// set up channel on which to send accepted connections
	listen := make(chan net.Conn, 100)
	go acceptConnection(listener, listen)

	ffmpegutil.Server()

	// loop work cycle with accept connections or interrupt
	// by system signal
	for {
		select {
		case conn := <-listen:
			go handleClient(conn)
		case killSignal := <-interrupt:
			ffmpegutil.WriteLog("Got signal:", killSignal.String())
			ffmpegutil.WriteLog("Stoping listening on ", listener.Addr().String())
			listener.Close()
			if killSignal == os.Interrupt {
				return "Daemon was interruped by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}

	// never happen, but need to complete code
	return usage, nil
}

// Accept a client connection and collect it in a channel
func acceptConnection(listener net.Listener, listen chan<- net.Conn) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		listen <- conn
	}
}

func handleClient(client net.Conn) {
	for {
		buf := make([]byte, 4096)
		numbytes, err := client.Read(buf)
		if numbytes == 0 || err != nil {
			return
		}
		client.Write(buf[:numbytes])
	}
}
