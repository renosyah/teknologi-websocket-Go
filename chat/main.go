package main

import "github.com/gorilla/mux"
import "net/http"
import "fmt"
import "log"
import "sync"
import "time"
import "github.com/gorilla/websocket"
import "gopkg.in/mgo.v2"
import "gopkg.in/mgo.v2/bson"
import "os"
import "html/template"

var router = mux.NewRouter()

func connect() *mgo.Session {
	var session, err = mgo.Dial("localhost")
	if err != nil {
		os.Exit(0)
	}
	return session
}

type akun struct {
	Nama     string `bson:"nama"`
	Username string `bson:"username"`
	Password string `bson:"password"`
}

type pesan struct {
	Pengirim string `bson:"pengirim"`
	Pesan    string `bson:"pesan"`
	Penerima string `bson:"penerima"`
}

type halaman_utama struct {
	Nama  string
	Index []akun
}

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

func (c *connection) reader(wg *sync.WaitGroup, wsconn *websocket.Conn, tujuan string, pengirim string) {
	defer wg.Done()
	session := connect()
	defer session.Close()

	var collection = session.DB("chat_app").C("chat_log")

	for {
		_, msg, err := wsconn.ReadMessage()
		if err != nil {
			break
		}
		data := pesan{pengirim, string(msg), tujuan}
		collection.Insert(data)

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

func chat(res http.ResponseWriter, req *http.Request) {
	tujuan := req.FormValue("nama_penerima")

	halaman, _ := template.ParseFiles("chat.html")
	data := map[string]string{
		"Nama": tujuan,
	}
	halaman.Execute(res, data)
}
func login(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "login.html")
}
func daftar(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "daftar.html")
}
func index(res http.ResponseWriter, req *http.Request) {
	akses := namauser(req)

	if akses != "" {

		session := connect()
		defer session.Close()

		var data_akun []akun

		var collection = session.DB("chat_app").C("akun")

		err := collection.Find(bson.M{"username": bson.M{"$ne": akses}}).All(&data_akun)
		if err != nil {
			fmt.Println("gagal mengambil data")
		}
		data := halaman_utama{akses, data_akun}

		halaman, _ := template.ParseFiles("index.html")
		halaman.Execute(res, data)

	} else {
		http.Redirect(res, req, "/", 301)
	}
}

func (wsh wsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	pengirim := namauser(req)

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
	go c.reader(&wg, wsconn, "", pengirim)
	wg.Wait()
	wsconn.Close()

}

func main() {
	h := newHub()

	router.HandleFunc("/chat", chat)
	router.HandleFunc("/index", index)
	router.HandleFunc("/daftar", daftar)
	router.HandleFunc("/mau_daftar", mau_daftar)
	router.HandleFunc("/", login)
	router.HandleFunc("/mau_login", mau_login)
	router.HandleFunc("/mau_logout", mau_logout)
	router.Handle("/ws", wsHandler{h: h})
	http.Handle("/", router)

	fmt.Println("running the server now....")
	http.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("data"))))
	http.ListenAndServe(":8080", nil)

}
