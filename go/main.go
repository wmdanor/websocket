package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/wmdanor/websocket/go/websocket"
	"github.com/wmdanor/websocket/go/websocket/frame"
)

type Websocket struct{}

func handler(w http.ResponseWriter, req *http.Request) {
	c, err := websocket.NewConnection(w, req)
	if err != nil {
		fmt.Printf("opening connection failed: %s\n", err.Error())
		if errors.Is(err, websocket.ErrInvalidHandshakeRequest) {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		w.Write([]byte(err.Error()))
		return
	}

	fmt.Printf("connection opened\n")

	defer c.Close()

	for {
		rFrame, err := c.ReadFrame()
		if err != nil {
			fmt.Printf("error reading frame from connection: %s\n", err.Error())
			return
		}

		fmt.Printf("Frame: %+v\n", rFrame)
		fmt.Printf("Data: %s\n", rFrame.ApplicationData)

		wPayload := []byte("All good")
		wFrame := frame.Frame{
			FIN:             frame.FINFinalFrame,
			Opcode:          frame.OpcodeTextFrame,
			PayloadLength:   uint64(len(wPayload)),
			ApplicationData: wPayload,
		}

		fmt.Printf("Sending: %+v\n", wFrame)
		err = c.WriteFrame(&wFrame)
		if err != nil {
			fmt.Printf("error writing frame to connection: %s\n", err.Error())
			return
		}
		fmt.Printf("Frame sent\n")

		err = c.Close()
		if err != nil {
			fmt.Printf("error closing connection: %s\n", err.Error())
			return
		}
	}
}

func main() {
	fmt.Printf("Starting server\n")

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 9001
	}

	addr := fmt.Sprintf(":%d", port)

	fmt.Printf("Starting server on addr: %s\n", addr)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(addr, nil))
}
