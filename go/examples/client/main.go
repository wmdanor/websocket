package main

import (
	"log/slog"
	"net/url"
	"os"

	websocket "github.com/wmdanor/websocket/go"
)

var (
	dialer websocket.Dialer = websocket.Dialer{}
)

func main() {
	setupLogger()

	u := url.URL{Scheme: "ws", Host: "localhost:9001", Path: ""}

	c, err := dialer.Dial(u.String(), nil)
	if err != nil {
		slog.Error("Opening connection failed", "err", err)
		return
	}

	defer func() {
		slog.Info("CLosing connection")
		err := c.Close()
		if err != nil {
			slog.Error("Failed to close connection", "err", err)
			return
		}
		slog.Info("Connection closed")
	}()

	slog.Info("Writing message")
	err = c.WriteMessage(websocket.TextMessage, []byte("Hello"))
	if err != nil {
		slog.Error("Failed to write message", "err", err)
		return
	}
	slog.Info("Message written")

	slog.Info("Reading message")
	_, data, err := c.NextMessage()
	if err != nil {
		slog.Error("Failed to read message", "err", err)
		return
	}
	slog.Info("Got message", "strdata", string(data))
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
		dialer.InternalLogger = l
	}

	slog.SetDefault(l)
}
