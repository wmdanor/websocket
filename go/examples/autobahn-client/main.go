package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	websocket "github.com/wmdanor/websocket/go"
)

const (
	testCountUrl     = "ws://localhost:9001/getCaseCount"
	runTestUrlF      = "ws://localhost:9001/runCase?case=%d&agent=ws"
	updateReportsUrl = "ws://localhost:9001/updateReports?agent=ws"
)

var (
	dialer websocket.Dialer = websocket.Dialer{}

	testCount   int = 0
	currentTest int = 1
)

func runNextTest(test int) {
	l := slog.With("testNumber", test)
	runTestUrl := fmt.Sprintf(runTestUrlF, test)

	c, err := dialer.Dial(runTestUrl, nil)
	if err != nil {
		l.Error("Opening connection failed", "err", err)
		return
	}

	for {
		l.Info("Getting next reader")
		mt, r, err := c.NextReader()
		if err != nil {
			l.Error("Failed to get next reader", "err", err)
			return
		}
		l.Info("Received message", "messageType", mt)

		l.Info("Getting next writer")
		w, err := c.NextWriter(mt)
		if err != nil {
			l.Error("Failed to get next writer", "err", err)
			return
		}
		l.Info("Created next writer")

		_, err = io.Copy(w, r)
		if err != nil {
			l.Error("Failed to echo message back", "err", err)
			return
		}

		err = w.Close()
		if err != nil {
			l.Error("Failed to close message writer", "err", err)
			return
		}
	}
}

func main() {
	setupLogger()

	slog.Error("Opening connection to test count url")
	c, err := dialer.Dial(testCountUrl, nil)
	if err != nil {
		slog.Error("Opening connection to test count url failed", "err", err)
		return
	}
	slog.Info("Reading test count message")
	_, data, err := c.NextMessage()
	c.Close()
	if err != nil {
		slog.Error("Failed to get test count message", "err", err)
		return
	}
	slog.Info("Received test count message", "strdata", string(data))
	testCount, err = strconv.Atoi(string(data))
	if err != nil {
		slog.Error("Failed to parse test count", "err", err)
		return
	}

	slog.Info("Running tests")

	for ; currentTest <= testCount; currentTest++ {
		runNextTest(currentTest)
	}

	slog.Error("Opening connection to update reports url")
	c, err = dialer.Dial(updateReportsUrl, nil)
	if err != nil {
		slog.Error("Opening connection to update reports url failed", "err", err)
		return
	}
	c.Close()

	slog.Info("Finished running tests")
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
