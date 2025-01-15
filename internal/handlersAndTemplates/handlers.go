package handlersAndTemplates

import (
	"WarehouseManager/internal/auth"
	"WarehouseManager/internal/model"
	"encoding/base64"
	"errors"
	"github.com/gorilla/mux"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var htmlFiles = []string{
	"account.html", "home.html", "login.html", "register.html", "warehouse.html", "warehouses.html", "items.html", "item.html",
	"items_search.html", "warehouses_search.html", "not_found.html"}
var templates = template.Must(template.ParseFiles(func() (result []string) {
	for _, file := range htmlFiles {
		result = append(result, "templates/"+file)
	}
	return result
}()...))
var sessionManager, _ = auth.LoadUserManager(func(name string) (model.WarehouseRepository, error) {
	return model.NewGORMSQLiteWarehouseRepository(name)
})

var router = mux.NewRouter()

type userSession struct {
	alias    string
	id       uint
	expireAt time.Time
}

var userSessions = map[string]userSession{}

func (s userSession) isExpired() bool {
	return s.expireAt.Before(time.Now())
}

func RefreshHandler(nextHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie("session_token")
		oldAlias := userSessions[c.Value].alias
		oldID := userSessions[c.Value].id
		createNewSession(w, oldAlias, oldID)
		delete(userSessions, c.Value)
		nextHandler.ServeHTTP(w, r)
		return
	}
}

func createNewSession(w http.ResponseWriter, alias string, userID uint) {
	newSessionToken := base64.URLEncoding.EncodeToString([]byte(auth.ShaHashing(strconv.Itoa(rand.Intn(1000000)))))
	newExpiration := time.Now().Add(time.Minute * 5)
	userSessions[newSessionToken] = userSession{
		alias,
		userID,
		newExpiration,
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: newExpiration,
	})
}

func SessionIsAbsentRedirectHandler(nextHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		var msg []byte
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				msg = []byte("You haven't logged in yet!")
				c := http.Cookie{
					Name:  "flash",
					Value: base64.URLEncoding.EncodeToString(msg),
				}
				http.SetCookie(w, &c)
				http.Redirect(w, r, `/login`, http.StatusFound)
				return
			} else {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
		}
		session, ok := userSessions[c.Value]
		if !ok || session.isExpired() {
			msg = []byte("Your previous session expired! Login again.")
			c := http.Cookie{
				Name:  "flash",
				Value: base64.URLEncoding.EncodeToString(msg),
			}
			http.SetCookie(w, &c)
			http.Redirect(w, r, `/login`, http.StatusFound)
			return
		}
		nextHandler.ServeHTTP(w, r)
		return
	}
}

type Page struct {
	LoggedIn     bool
	MessageTypeA string
	MessageTypeB string
	Error        string
}

type WarehousePage struct {
	Page
	Content []model.Warehouse
}

type ItemPage struct {
	Page
	Content []model.Item
}

func SessionIsPresentHandler(nextHandler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err == nil {
			session, ok := userSessions[c.Value]
			if ok && !session.isExpired() {
				msg := []byte("You already have an active session!")
				newC := http.Cookie{
					Name:  "flash",
					Value: base64.URLEncoding.EncodeToString(msg),
				}
				http.SetCookie(w, &newC)
				http.Redirect(w, r, `/home`, http.StatusFound)
				return
			}
		} else if !errors.Is(err, http.ErrNoCookie) {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		nextHandler(w, r)
	}
}

func SessionIsAbsentHomeHandler(nextHandler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				err1 := templates.ExecuteTemplate(w, "templates/home.html", Page{LoggedIn: false, MessageTypeA: "Login to begin using the app!"})
				if err1 != nil {
					http.Error(w, err1.Error(), http.StatusInternalServerError)
				}
				return
			} else {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
		}
		session, ok := userSessions[c.Value]
		if !ok || session.isExpired() {
			err1 := templates.ExecuteTemplate(w, "templates/home.html", Page{LoggedIn: false, MessageTypeA: "Previous session expired! Login again.!"})
			if err1 != nil {
				http.Error(w, err1.Error(), http.StatusInternalServerError)
			}
			return
		}
		nextHandler(w, r)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("session_token")
	session, _ := userSessions[c.Value]
	var page Page
	var err error
	page.LoggedIn = true
	page.MessageTypeA = processFlashMessage(w, r, "flash")
	page.MessageTypeB, err = evaluateItems(w, session)
	page.Error = err.Error()
	err2 := templates.ExecuteTemplate(w, "templates/home.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
}

func processFlashMessage(w http.ResponseWriter, r *http.Request, cookieName string) string {
	c2, err2 := r.Cookie(cookieName)
	if err2 != nil {
		if !errors.Is(err2, http.ErrNoCookie) {
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
	} else {
		flashMessage, err3 := base64.URLEncoding.DecodeString(c2.Value)
		if err3 != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
		newC := http.Cookie{
			Name:    cookieName,
			Value:   "",
			MaxAge:  -1,
			Expires: time.Unix(1, 0),
		}
		http.SetCookie(w, &newC)
		return string(flashMessage)
	}
	return ""
}

func evaluateItems(w http.ResponseWriter, session userSession) (string, error) {
	notification := "the following items are almost out of stock: "
	resupply := false
	items, err1 := sessionManager.ListAllItems(session.id)
	if err1 != nil {
		return "", err1
	}
	for _, v := range items {
		if v.Quantity < 100 {
			notification += v.Name + " "
			resupply = true
		}
	}
	if !resupply {
		notification = ""
	}
	return notification, nil
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	message := processFlashMessage(w, r, "error")
	err := templates.ExecuteTemplate(w, "templates/login.html", Page{Error: message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	message := processFlashMessage(w, r, "error")
	err := templates.ExecuteTemplate(w, "templates/login.html", Page{Error: message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	newUsername := r.FormValue("newUsername")
	newPassword1 := r.FormValue("newPassword1")
	newPassword2 := r.FormValue("newPassword2")
	if newPassword1 != newPassword2 {
		cookie := http.Cookie{
			Name:  "error",
			Value: base64.URLEncoding.EncodeToString([]byte("the passwords don't match")),
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	} else {
		err := sessionManager.Register(newUsername, newPassword1)
		if err != nil {
			cookie := http.Cookie{
				Name:  "error",
				Value: base64.URLEncoding.EncodeToString([]byte(err.Error())),
			}
			http.SetCookie(w, &cookie)
			http.Redirect(w, r, "/register", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	userID, err := sessionManager.Login(username, password)
	if err != nil {
		cookie := http.Cookie{
			Name:  "error",
			Value: base64.URLEncoding.EncodeToString([]byte(err.Error())),
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	createNewSession(w, username, userID)
	http.Redirect(w, r, "/home", http.StatusFound)
	return
}

func ManageUserSessions() {
	time.Sleep(30 * time.Second)
	expiredK := make([]string, 0)
	for k, v := range userSessions {
		if v.isExpired() {
			expiredK = append(expiredK, k)
		}
	}
	for _, k := range expiredK {
		err := sessionManager.Logout(userSessions[k].alias)
		if err != nil {
			panic(err)
		}
		delete(userSessions, k)
	}
}
