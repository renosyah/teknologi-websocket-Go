package main

import "net/http"
import "fmt"
import "log"
import "sync"
import "time"
import "github.com/gorilla/websocket"
import "flag"

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type hub struct {
	ConnectionMx sync.RWMutex
	Connection   map[*connection]struct{}
	Broadcast    chan []byte
	LogMx        sync.RWMutex
	log          [][]byte
}

type connection struct {
	send chan []byte
	h    *hub
}

type wsHandler struct {
	h *hub
}

func (c *connection) reader(wg *sync.WaitGroup, wsconn *websocket.Conn) {
	defer wg.Done()

	for {
		_, msg, err := wsconn.ReadMessage()
		if err != nil {
			break
		}
		c.h.Broadcast <- msg
	}
}

func (c *connection) write(wg *sync.WaitGroup, wsconn *websocket.Conn) {
	defer wg.Done()

	for msg := range c.send {
		err := wsconn.WriteMessage(websocket.TextMessage, msg)

		if err != nil {
			break
		}
	}

}

func newHub() *hub {
	h := &hub{
		ConnectionMx: sync.RWMutex{},
		Broadcast:    make(chan []byte),
		Connection:   make(map[*connection]struct{}),
	}
	go func() {
		for {
			msg := <-h.Broadcast
			h.ConnectionMx.RLock()
			for c := range h.Connection {
				select {
				case c.send <- msg:

				case <-time.After((1 * time.Second)):
					log.Printf("shutting down connection %s", c)
					h.removeconnection(c)
				}
			}
			h.ConnectionMx.RUnlock()
		}

	}()
	return h
}

func (h *hub) addconnection(conn *connection) {
	h.ConnectionMx.Lock()
	defer h.ConnectionMx.Unlock()
	h.Connection[conn] = struct{}{}
}

func (h *hub) removeconnection(conn *connection) {
	h.ConnectionMx.Lock()
	defer h.ConnectionMx.Unlock()
	if _, ok := h.Connection[conn]; ok {
		delete(h.Connection, conn)
		close(conn.send)
	}
}

func home(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "index.html")
}

func (wsh wsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	wsconn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Printf("error upgrading %s", err)
		return
	}
	c := &connection{send: make(chan []byte, 256), h: wsh.h}
	c.h.addconnection(c)
	defer c.h.removeconnection(c)

	var wg sync.WaitGroup
	wg.Add(2)

	go c.write(&wg, wsconn)
	go c.reader(&wg, wsconn)
	wg.Wait()
	wsconn.Close()

}

func main() {
	flag.Parse()
	h := newHub()
	router := http.NewServeMux()

	router.HandleFunc("/", home)
	router.Handle("/ws", wsHandler{h: h})
	fmt.Println("running the server now....")
	http.ListenAndServe(":8080", router)

}
