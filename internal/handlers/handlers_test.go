package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"
)

var rr = httptest.NewRecorder()

func TestHandlers(t *testing.T) {
	t.Cleanup(func() {
		clean(t)
	})
	Methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	homeURLs := []string{"/", "/home"}
	var rr1 *httptest.ResponseRecorder
	for _, homeURL := range homeURLs {
		for _, method := range Methods {
			t.Run("Routing tests "+method+" home expired session cookies path "+homeURL, func(t *testing.T) {
				t.Cleanup(func() {
					SetSessions(make(map[string]userSession))
					rr = httptest.NewRecorder()
				})
				router := BuildAPPRouter("templates/")
				req, err := http.NewRequest(method, homeURL, nil)
				if err != nil {
					t.Fatalf("Reported error: " + err.Error())
				}
				activeSessions := map[string]userSession{
					"10": {
						alias:    "david",
						id:       0,
						expireAt: time.Unix(1, 0),
					},
				}
				SetSessions(activeSessions)
				req.AddCookie(&http.Cookie{Name: "session_token", Value: "10"})
				router.ServeHTTP(rr, req)
				if method != http.MethodGet {
					if rr.Code != http.StatusMethodNotAllowed && rr.Body.String() == "Method not allowed" {
						t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusOK, rr.Code)
					}
				} else {
					if rr.Code != http.StatusOK {
						t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusOK, rr.Code)
					}
				}
			})
		}
	}
	for _, homeURL := range homeURLs {
		for _, method := range Methods {
			if method != http.MethodGet {
				t.Run("Routing tests "+method+" home valid session cookies path "+homeURL, func(t *testing.T) {
					t.Cleanup(func() {
						SetSessions(make(map[string]userSession))
						rr = httptest.NewRecorder()
					})
					router := BuildAPPRouter("templates/")
					req, err := http.NewRequest(method, homeURL, nil)
					if err != nil {
						t.Fatalf("Reported error: " + err.Error())
					}
					activeSessions := map[string]userSession{
						"10": {
							alias:    "david",
							id:       0,
							expireAt: time.Now().Add(time.Minute * 10),
						},
					}
					SetSessions(activeSessions)
					req.AddCookie(&http.Cookie{Name: "session_token", Value: "10"})
					router.ServeHTTP(rr, req)
					if rr.Code != http.StatusMethodNotAllowed && rr.Body.String() == "Method not allowed" {
						t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusOK, rr.Code)
					}

				})
			}
		}
	}
	urls := []string{
		"/items",
		"/item/10",
		"/warehouses",
		"/warehouse/2",
		"/item/3/consume",
		"/item/4/transfer",
		"/item/5/supply",
		"/items/search",
		"/warehouses/search",
		"/account",
	}
	for _, homeURL := range urls {
		t.Run("routing tests redirect to login page path "+homeURL, func(t *testing.T) {
			rr = httptest.NewRecorder()
			router := BuildAPPRouter("templates/")
			req, err := http.NewRequest(http.MethodGet, homeURL, nil)
			if err != nil {
				t.Fatalf("Reported error: " + err.Error())
			}
			activeSessions := map[string]userSession{
				"abc": {
					alias:    "david",
					id:       0,
					expireAt: time.Unix(1, 0),
				},
			}
			SetSessions(activeSessions)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: "10"})
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d\nBody: %v", http.StatusFound, rr.Code, rr.Body)
			}
			if rr.Header().Get("Location") != "/login" {
				t.Errorf("Redirected to wrong URL. Expected /login, got %s", rr.Header().Get("Location"))
			}
		})
	}
	urls = []string{
		"/login",
		"/register",
	}
	for _, v := range urls {
		t.Run("routing tests redirect to home page path "+v, func(t *testing.T) {
			rr = httptest.NewRecorder()
			router := BuildAPPRouter("templates/")
			req, err := http.NewRequest(http.MethodGet, v, nil)
			if err != nil {
				t.Fatalf("Reported error: " + err.Error())
			}
			activeSessions := map[string]userSession{
				"abc": {
					alias:    "david",
					id:       0,
					expireAt: time.Now().Add(10 * time.Minute),
				},
			}
			SetSessions(activeSessions)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: "abc"})
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/" && rr.Header().Get("Location") != "/home" {
				t.Errorf("Redirected to wrong URL. Expected / or /home, got %s", rr.Header().Get("Location"))
			}
		})
	}
	SetSessions(make(map[string]userSession))
	t.Run("Registration login and logout", func(t *testing.T) {
		req1, err1 := http.NewRequest(http.MethodPost, "/register", nil)
		if err1 != nil {
			t.Fatalf("Reported error: " + err1.Error())
		}
		form, _ := url.ParseQuery(req1.URL.RawQuery)
		form.Add("newUsername", "david")
		form.Add("newPassword1", "longPassword")
		form.Add("newPassword2", "longPassword")
		req1.URL.RawQuery = form.Encode()
		router := BuildAPPRouter("templates/")
		router.ServeHTTP(rr, req1)
		if rr.Code != http.StatusFound {
			t.Fatalf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
		}
		if rr.Header().Get("Location") != "/login" {
			t.Errorf("Redirected to wrong URL. Expected /login, got %s", rr.Header().Get("Location"))
		}
		rr, _ = login(t, router, "longPassword")
		if rr.Code != http.StatusFound {
			t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
		}
		if rr.Header().Get("Location") != "/" && rr.Header().Get("Location") != "/home" {
			t.Errorf("Redirected to wrong URL. Expected /, got %s", rr.Header().Get("Location"))
		}
		neededCookies := rr.Result().Cookies()
		t.Run("Redirect session present", func(t *testing.T) {
			req3, err3 := http.NewRequest(http.MethodGet, "/register", nil)
			if err3 != nil {
				t.Fatalf("Reported error: " + err3.Error())
			}
			for _, cookie := range neededCookies {
				req3.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			router.ServeHTTP(rr, req3)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/home" {
				t.Errorf("Redirected to wrong URL. Expected /home, got %s", rr.Header().Get("Location"))
			}
			req4, err4 := http.NewRequest(http.MethodGet, "/login", nil)
			if err4 != nil {
				t.Fatalf("Reported error: " + err4.Error())
			}
			for _, cookie := range neededCookies {
				req4.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			router.ServeHTTP(rr, req4)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/home" {
				t.Errorf("Redirected to wrong URL. Expected /home, got %s", rr.Header().Get("Location"))
			}
		})
		t.Run("Logout", func(t *testing.T) {
			req3, err3 := http.NewRequest(http.MethodPost, "/logout", nil)
			if err3 != nil {
				t.Fatalf("Reported error: " + err3.Error())
			}
			for _, cookie := range neededCookies {
				req3.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			router.ServeHTTP(rr, req3)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d\nError: %v", http.StatusFound, rr.Code, rr.Body)
			}
			if rr.Header().Get("Location") != "/" && rr.Header().Get("Location") != "/home" {
				t.Errorf("Redirected to wrong URL. Expected /, got %s", rr.Header().Get("Location"))
			}
		})
	})
	urls = []string{
		"/uuuuuuuuuuu",
		"/haha",
		"/veryFunny",
		"/testingIsVeryFunny...",
	}
	for _, v := range urls {
		t.Run("Routing Not found", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, v, nil)
			if err != nil {
				t.Fatalf("Reported error: " + err.Error())
			}
			rr = httptest.NewRecorder()
			router := BuildAPPRouter("templates/")
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("Returned wrong status code. Expected %d, got %d", http.StatusNotFound, rr.Code)
			}
		})
	}
	t.Run("Password change", func(t *testing.T) {
		router := BuildAPPRouter("templates/")
		rr1, _ = login(t, router, "longPassword")
		req, err := http.NewRequest(http.MethodPost, "/account", nil)
		if err != nil {
			t.Fatalf("Reported error: " + err.Error())
		}
		neededCookies := rr1.Result().Cookies()
		for _, cookie := range neededCookies {
			req.AddCookie(cookie)
		}
		rr = httptest.NewRecorder()
		form, _ := url.ParseQuery(req.URL.RawQuery)
		form.Add("oldPassword", "longPassword")
		form.Add("newPassword", "shortPassword")
		req.URL.RawQuery = form.Encode()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Errorf("Returned wrong status code. Expected %d, got %d\nError: %v", http.StatusFound, rr.Code, rr.Body)
		}
		if rr.Header().Get("Location") != "/home" {
			t.Errorf("Redirected to wrong URL. Expected /home, got %s", rr.Header().Get("Location"))
		}
	})
	t.Run("Basic Operations on resource", func(t *testing.T) {
		router := BuildAPPRouter("templates/")
		t.Run("Creating and updating a resource", func(t *testing.T) {
			for i := 1; i < 3; i++ {
				req, err := http.NewRequest(http.MethodPost, "/warehouses", nil)
				if err != nil {
					t.Fatalf("Reported error: " + err.Error())
				}
				neededCookies := rr1.Result().Cookies()
				for _, cookie := range neededCookies {
					req.AddCookie(cookie)
				}
				rr = httptest.NewRecorder()
				form, _ := url.ParseQuery(req.URL.RawQuery)
				form.Add("warehousePosition", "Tuscany")
				form.Add("warehouseName", "La Rosa "+strconv.Itoa(i))
				form.Add("warehouseCapacity", "2000")
				req.URL.RawQuery = form.Encode()
				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusFound {
					t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
				}
				if rr.Header().Get("Location") != "/warehouse/"+strconv.Itoa(i) {
					t.Errorf("Wrong redirect- expected redirect: /warehouse/%d actual redirect: %s", i, rr.Header().Get("Location"))
				}
			}

			req2, err2 := http.NewRequest(http.MethodPost, "/items", nil)
			if err2 != nil {
				t.Fatalf("Reported error: " + err2.Error())
			}
			neededCookies := rr1.Result().Cookies()
			for _, cookie := range neededCookies {
				req2.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			form, _ := url.ParseQuery(req2.URL.RawQuery)
			form.Add("itemName", "Keyboard")
			form.Add("itemCategory", "electronics")
			form.Add("itemDescription", "logitech keyboard")
			req2.URL.RawQuery = form.Encode()
			router.ServeHTTP(rr, req2)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/item/1" {
				t.Errorf("Wrong redirect- expected redirect: /item/1 actual redirect: %s", rr.Header().Get("Location"))
			}
		})
		t.Run("Supplying transferring removing resources", func(t *testing.T) {
			req2, err2 := http.NewRequest(http.MethodPost, "/item/1/supply", nil)
			if err2 != nil {
				t.Fatalf("Reported error: " + err2.Error())
			}
			neededCookies := rr1.Result().Cookies()
			for _, cookie := range neededCookies {
				req2.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			form, _ := url.ParseQuery(req2.URL.RawQuery)
			form.Add("amount", "10")
			form.Add("warehouseID", "1")
			req2.URL.RawQuery = form.Encode()
			router.ServeHTTP(rr, req2)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/items" {
				t.Errorf("Wrong redirect- expected redirect: /items actual redirect: %s", rr.Header().Get("Location"))
			}
			req3, err3 := http.NewRequest(http.MethodPost, "/item/1/transfer", nil)
			if err3 != nil {
				t.Fatalf("Reported error: " + err3.Error())
			}
			neededCookies = rr1.Result().Cookies()
			for _, cookie := range neededCookies {
				req3.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			form, _ = url.ParseQuery(req3.URL.RawQuery)
			form.Add("amount", "1")
			form.Add("srcID", "1")
			form.Add("destID", "2")
			req3.URL.RawQuery = form.Encode()
			router.ServeHTTP(rr, req3)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/items" {
				t.Errorf("Wrong redirect- expected redirect: /items actual redirect: %s", rr.Header().Get("Location"))
			}
			req4, err4 := http.NewRequest(http.MethodPost, "/item/1/consume", nil)
			if err4 != nil {
				t.Fatalf("Reported error: " + err4.Error())
			}
			neededCookies = rr1.Result().Cookies()
			for _, cookie := range neededCookies {
				req4.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			form, _ = url.ParseQuery(req4.URL.RawQuery)
			form.Add("amount", "5")
			form.Add("warehouseID", "1")
			req4.URL.RawQuery = form.Encode()
			router.ServeHTTP(rr, req4)
			if rr.Code != http.StatusFound {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusFound, rr.Code)
			}
			if rr.Header().Get("Location") != "/items" {
				t.Errorf("Wrong redirect- expected redirect: /items actual redirect: %s\nError: %v", rr.Header().Get("Location"), rr.Body)
			}
		})
		t.Run("Search Operations", func(t *testing.T) {
			req2, err2 := http.NewRequest(http.MethodPost, "/warehouses/search", nil)
			if err2 != nil {
				t.Fatalf("Reported error: " + err2.Error())
			}
			neededCookies := rr1.Result().Cookies()
			for _, cookie := range neededCookies {
				req2.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			form, _ := url.ParseQuery(req2.URL.RawQuery)
			form.Add("warehousePosition", "Tuscany")
			req2.URL.RawQuery = form.Encode()
			router.ServeHTTP(rr, req2)
			if rr.Code != http.StatusOK {
				t.Errorf("Returned wrong status code. Expected %d, got %d", http.StatusOK, rr.Code)
			}
		})
	})
	urls = []string{
		"/items",
		"/item/1",
		"/warehouses",
		"/warehouse/1",
		"/items/search",
		"/warehouses/search",
		"/account",
	}
	for _, homeURL := range urls {
		t.Run("get various resource pages "+homeURL, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, homeURL, nil)
			neededCookies := rr1.Result().Cookies()
			for _, cookie := range neededCookies {
				req.AddCookie(cookie)
			}
			rr = httptest.NewRecorder()
			router := BuildAPPRouter("templates/")
			if err != nil {
				t.Fatalf("Reported error: " + err.Error())
			}
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("Returned wrong status code. Expected %d, got %d\nBody: %v", http.StatusOK, rr.Code, rr.Body)
			}
		})
	}
}

func login(t *testing.T, router *mux.Router, password string) (*httptest.ResponseRecorder, *http.Request) {
	req, err2 := http.NewRequest(http.MethodPost, "/login", nil)
	rr = httptest.NewRecorder()
	if err2 != nil {
		t.Fatalf("Reported error: " + err2.Error())
	}
	form, _ := url.ParseQuery(req.URL.RawQuery)
	form.Add("username", "david")
	form.Add("password", password)
	req.URL.RawQuery = form.Encode()
	router.ServeHTTP(rr, req)
	return rr, req
}

func clean(t *testing.T) {
	SetSessions(make(map[string]userSession))
	rr = httptest.NewRecorder()
	manager := GetManager()
	err := manager.DeleteAllUsers()
	if err != nil {
		t.Fatalf("Reported error: %v", err)
	}
	dirEntries, err1 := os.ReadDir("./data")
	if err1 != nil {
		t.Fatalf("Failed to read directory\nerror: %v", err1)
	}
	for _, v := range dirEntries {
		matching, err2 := regexp.MatchString(`usr[0-9]+\.db`, v.Name())
		if err2 != nil {
			t.Fatalf("Failed to match file name\nerror: %v", err2)
		}
		if matching {
			err3 := os.Remove("data/" + v.Name())
			if err3 != nil {
				t.Fatalf("Failed to remove file\nerror: %v", err3)
			}
		}
		_ = os.Remove("users.json")
	}
}
