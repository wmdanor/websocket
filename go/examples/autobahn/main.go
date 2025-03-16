package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	websocket "github.com/wmdanor/websocket/go"
)

var (
	upgrader websocket.Upgrader = websocket.Upgrader{}
)

func handler(w http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(w, req)
	if err != nil {
		slog.Error("Opening connection failed", "err", err)
		if errors.Is(err, websocket.ErrHandshakeFailure) {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		w.Write([]byte(err.Error()))
		return
	}

	defer c.Close()

	for {
		slog.Info("Getting next reader")
		mt, r, err := c.NextReader()
		if err != nil {
			slog.Error("Failed to get next reader", "err", err)
			return
		}
		slog.Info("Received message", "messageType", mt)

		slog.Info("Getting next writer")
		w, err := c.NextWriter(mt)
		if err != nil {
			slog.Error("Failed to get next writer", "err", err)
			return
		}
		slog.Info("Created next writer")

		_, err = io.Copy(w, r)
		if err != nil {
			slog.Error("Failed to echo message back", "err", err)
			return
		}

		err = w.Close()
		if err != nil {
			slog.Error("Failed to close message writer", "err", err)
			return
		}
	}
}

func main() {
	setupLogger()

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 9001
	}

	addr := fmt.Sprintf(":%d", port)

	slog.Info("Starting server", "addr", addr)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func setupLogger() {
	l := slog.New(slog.DiscardHandler)
	if os.Getenv("LOG") != "1" {
		slog.SetDefault(l)
		return
	}

	logDest := os.Stdout
	if os.Getenv("LOG_FILE") != "" {
		f, err := os.Create(os.Getenv("WS_LOG_FILE"))
		if err != nil {
			panic(err)
		}
		logDest = f
	}

	l = slog.New(slog.NewTextHandler(logDest, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	if os.Getenv("WS_LOG") == "1" {
		upgrader.InternalLogger = l
	}

	slog.SetDefault(l)
}
