package main

import "github.com/gorilla/securecookie"
import "net/http"
import "gopkg.in/mgo.v2/bson"
import "fmt"

var cookiehandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func setsesi(nama_user string, res http.ResponseWriter) {
	value := map[string]string{
		"name": nama_user,
	}
	if encoded, err := cookiehandler.Encode("session", value); err == nil {
		cookie_ku := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(res, cookie_ku)
	}
}

func namauser(req *http.Request) (name_usernya string) {
	if cookie_ini, err := req.Cookie("session"); err == nil {
		nilai_cookie := make(map[string]string)
		if err = cookiehandler.Decode("session", cookie_ini.Value, &nilai_cookie); err == nil {
			name_usernya = nilai_cookie["name"]
		}

	}
	return name_usernya
}

func clear_session(res http.ResponseWriter) {
	bersihkan_cookie_ku := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(res, bersihkan_cookie_ku)
}

func mau_login(res http.ResponseWriter, req *http.Request) {
	username := req.FormValue("username")
	passwd := req.FormValue("password")

	akses := akun{}
	jalur := "/"

	session := connect()
	defer session.Close()

	var collection = session.DB("chat_app").C("akun")
	collection.Find(bson.M{"username": username}).One(&akses)

	if akses.Password == passwd {
		jalur = "/index"
		setsesi(akses.Username, res)

	}
	http.Redirect(res, req, jalur, 302)
}

func mau_daftar(res http.ResponseWriter, req *http.Request) {
	session := connect()
	defer session.Close()

	nama := req.FormValue("nama")
	username := req.FormValue("username")
	passwd := req.FormValue("password")

	data_akun := akun{Nama: nama, Username: username, Password: passwd}

	var collection = session.DB("chat_app").C("akun")
	err := collection.Insert(data_akun)
	if err != nil {
		fmt.Println("gagal mendaftar")
	}

	http.Redirect(res, req, "/", 301)

}
func mau_logout(res http.ResponseWriter, req *http.Request) {
	clear_session(res)
	http.Redirect(res, req, "/", 301)
}
