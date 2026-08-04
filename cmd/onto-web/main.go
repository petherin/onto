// Package main is the entry point for the Onto web UI. It wires the existing
// cli.App to the web.Server and opens a browser to the local address.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/petherin/onto/internal/interface/cli"
	"github.com/petherin/onto/internal/interface/web"
)

func main() {
	app := cli.NewApp()
	srv := web.NewServer(app)

	port := "8080"
	if p := os.Getenv("ONTO_PORT"); p != "" {
		port = p
	}
	addr := "127.0.0.1:" + port

	// Find a free port if the default is busy.
	if ln, err := net.Listen("tcp", addr); err != nil {
		// Try a random free port.
		ln2, err2 := net.Listen("tcp", "127.0.0.1:0")
		if err2 != nil {
			log.Fatalf("cannot bind: %v", err2)
		}
		addr = ln2.Addr().String()
		_ = ln2.Close()
	} else {
		_ = ln.Close()
	}

	url := "http://" + addr
	fmt.Printf("Onto Explorer  →  %s\n", url)
	fmt.Println("Press Ctrl+C to stop.")

	go func() {
		time.Sleep(300 * time.Millisecond)
		openBrowser(url)
	}()

	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
