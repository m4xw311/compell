package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	argsWithoutCommand := os.Args[1:]
	http.HandleFunc("/ws", handleWS(argsWithoutCommand))

	fmt.Println("WebSocket server running on ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWS(cmdArgs []string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer conn.Close()

		// Start your agent subprocess
		// Replace "./your-agent" with the actual binary or script
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Println("Error getting stdin:", err)
			return
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Println("Error getting stdout:", err)
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Println("Error getting stderr:", err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Println("Error starting agent:", err)
			return
		}

		// Pipe agent stdout → WebSocket
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				message := fmt.Sprintf("{\"type\": \"stdout\", \"data\": \"%s\"}", line)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
					log.Println("WS write error:", err)
					return
				}
			}
		}()

		// Pipe agent stderr → WebSocket
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				message := fmt.Sprintf("{\"type\": \"stderr\", \"data\": \"%s\"}", line)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
					log.Println("WS write error:", err)
					return
				}
			}
		}()

		// Pipe WebSocket messages → agent stdin
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("WS read error:", err)
				return
			}
			_, err = stdin.Write(append(msg, '\n'))
			if err != nil {
				log.Println("Stdin write error:", err)
				return
			}
		}
	}
}
