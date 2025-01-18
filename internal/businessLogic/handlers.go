package businessLogic

import (
	"WarehouseManager/internal/auth"
	"WarehouseManager/internal/model"
	"encoding/base64"
	"errors"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// The various html files used in this project
var htmlFiles = []string{
	"account.html", "home.html", "login.html", "register.html", "warehouse.html", "warehouses.html", "items.html", "item.html",
	"items_search.html", "warehouses_search.html", "not_found.html", "navbar.html", "notifications.html"}

// We complete the file path by appending the "templates/" prefix and parse them to generate a template file
var templates = template.Must(template.ParseFiles(func() (result []string) {
	for _, file := range htmlFiles {
		result = append(result, "templates/"+file)
	}
	return result
}()...))

// An instance of AuthenticationManager from the auth package
var authManager, _ = auth.LoadAuthManager(func(name string) (model.WarehouseRepository, error) {
	return model.NewGORMSQLiteWarehouseRepository(name)
})

// GetManager is a method only used for testing
func GetManager() *auth.AuthenticationManager {
	return authManager
}

// userSession allows the user to access the web APP until its expiration if not refreshed
type userSession struct {
	alias    string
	id       uint
	expireAt time.Time
}

// global variable to manage the sessions of clients
var userSessions = make(map[string]userSession)

// SetSessions is a method used only for testing
func SetSessions(sessions map[string]userSession) {
	userSessions = sessions
}

// isExpired is a utility function to check if a session is expired or not
func (s userSession) isExpired() bool {
	return s.expireAt.Before(time.Now())
}

// Page represents the structure of a generic page of the web APP
type Page struct {
	// boolean expressing whether the user is logged in or not
	LoggedIn bool
	// authentication related flash messages
	AuthMsg string
	// application related flash messages
	APPMsg string
	// application-related errors flash messages
	APPError string
	// logout errors
	LOError string
}

// WarehousesPage represents the particular page obtained by calling GET /warehouses
type WarehousesPage struct {
	Page
	Content []model.Warehouse
}

// similar to WarehousesPage
type ItemsPage struct {
	Page
	Content []model.Item
}

// ItemPage represents the particular page obtained by calling GET /item/{id:[0-9]+}
type ItemPage struct {
	Page
	Item              model.Item
	HostingWarehouses []model.Warehouse
	AllWarehouses     []model.Warehouse
}

// similar to ItemPage
type WarehousePage struct {
	Page
	Warehouse model.Warehouse
}

// SearchPage display the result of a searching operation
type SearchPage struct {
	Page
	Warehouses []model.Warehouse
	Items      []model.Item
}

type AccountPage struct {
	Page
	Username string
}

// Middleware for HomeHandler. It helps in showing the right messages in case of an access without login
func SessionIsAbsentHomeHandler(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			c, err := r.Cookie("session_token")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
					err1 := templates.ExecuteTemplate(w, "home.html", &Page{LoggedIn: false, AuthMsg: "Login to begin using the app!"})
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
				err1 := templates.ExecuteTemplate(w, "home.html", &Page{LoggedIn: false, AuthMsg: "Previous session expired! Login again.!"})
				if err1 != nil {
					http.Error(w, err1.Error(), http.StatusInternalServerError)
				}
				return
			}
			nextHandler.ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

// HomeHandler displays the home page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := getSession(w, r)
	var page Page
	page.LoggedIn = true
	page.AuthMsg = processFlashMessage(w, r, "flash")
	page.LOError = processFlashMessage(w, r, "lOError")
	page.APPMsg = evaluateItems(session)
	err2 := templates.ExecuteTemplate(w, "home.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
}

// evaluateItems crafts a message notifying the user if some items in the inventory are running out
func evaluateItems(session userSession) string {
	notification := "the following items are almost out of stock: "
	resupply := false
	items, err1 := authManager.ListAllItems(session.id)
	if err1 != nil {
		return err1.Error()
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
	return notification
}

// SessionIsAbsentRedirectHandler Redirects the HTTP request to the login page if the user isn't logged in yet
func SessionIsAbsentRedirectHandler(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				setFlashMessage(w, "flash", "You haven't logged in yet!")
				http.Redirect(w, r, `/login`, http.StatusFound)
				return
			} else {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
		}
		session, ok := userSessions[c.Value]
		if !ok || session.isExpired() {
			setFlashMessage(w, "flash", "Your previous session expired! Login again.")
			http.Redirect(w, r, `/login`, http.StatusFound)
			return
		}
		nextHandler.ServeHTTP(w, r)
		return
	}
}

// RefreshHandler refreshes the session corresponding to the user making the request
func RefreshHandler(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if errors.Is(err, http.ErrNoCookie) {
			http.Error(w, "No session cookie when there should be", http.StatusInternalServerError)
		}
		oldAlias := userSessions[c.Value].alias
		oldID := userSessions[c.Value].id
		createNewSession(w, oldAlias, oldID, c.Value)
		nextHandler.ServeHTTP(w, r)
		return
	}
}

var tempCookieContainer *http.Cookie

// createNewSession allows for the creation of a new session object to substitute the old one
func createNewSession(w http.ResponseWriter, alias string, userID uint, oldToken string) {
	newSessionToken := base64.URLEncoding.EncodeToString([]byte(auth.ShaHashing(strconv.Itoa(rand.Intn(1000000)))))
	newExpiration := time.Now().Add(time.Minute * 5)
	userSessions[newSessionToken] = userSession{
		alias,
		userID,
		newExpiration,
	}
	tempCookieContainer = &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: newExpiration,
	}
	http.SetCookie(w, tempCookieContainer)
	if oldToken != "" {
		delete(userSessions, oldToken)
	}
}

// ItemsHandler handlers operations on the /items page
func ItemsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		session, ok := getSessionAfterRefresh()
		if !ok {
			http.Error(w, "no session found", http.StatusInternalServerError)
		}
		var page ItemsPage
		page.LoggedIn = true
		page.LOError = processFlashMessage(w, r, "lOError")
		items, err2 := authManager.ListAllItems(session.id)
		if err2 != nil {
			page.APPError = err2.Error()
			err3 := templates.ExecuteTemplate(w, "items.html", page)
			if err3 != nil {
				http.Error(w, err2.Error(), http.StatusInternalServerError)
			}
			return
		}
		page.APPError = processFlashMessage(w, r, "error")
		page.Content = items
		page.APPMsg = evaluateItems(session)
		err4 := templates.ExecuteTemplate(w, "items.html", page)
		if err4 != nil {
			http.Error(w, err4.Error(), http.StatusInternalServerError)
		}
		return
	} else if r.Method == http.MethodPost {
		session, ok := getSessionAfterRefresh()
		if !ok {
			http.Error(w, "no session found", http.StatusInternalServerError)
		}
		itemName := r.FormValue("itemName")
		itemCategory := r.FormValue("itemCategory")
		itemDescription := r.FormValue("itemDescription")
		createItem(w, r, session, itemName, itemCategory, itemDescription)
		return
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// createItem creates new items for the /items page
func createItem(w http.ResponseWriter, r *http.Request, session userSession, itemName string, itemCategory string, itemDescription string) {
	err3 := authManager.CreateItem(session.id, itemName, itemCategory, itemDescription)
	if err3 != nil {
		setFlashMessage(w, "error", err3.Error())
		http.Redirect(w, r, "/items", http.StatusFound)
		return
	}
	item, err4 := authManager.FindItemByName(session.id, itemName)
	if err4 != nil {
		http.Error(w, err4.Error(), http.StatusInternalServerError)
		return
	}
	if len(item) == 0 {
		http.Error(w, "Newly created item not found", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/item/"+strconv.Itoa(int(item[0].ID)), http.StatusFound)
	return
}

// WarehousesHandler handlers the operations on the /warehouses page
func WarehousesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var page WarehousesPage
		page.LoggedIn = true
		page.LOError = processFlashMessage(w, r, "lOError")
		session, ok := getSessionAfterRefresh()
		if !ok {
			http.Error(w, "no session found", http.StatusInternalServerError)
		}
		warehouses, err1 := authManager.ListAllWarehouses(session.id)
		if err1 != nil {
			page.APPError = err1.Error()
			err2 := templates.ExecuteTemplate(w, "warehouses.html", page)
			if err2 != nil {
				http.Error(w, err2.Error(), http.StatusInternalServerError)
			}
			return
		}
		page.Content = warehouses
		err2 := templates.ExecuteTemplate(w, "warehouses.html", page)
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
		}
		return
	} else if r.Method == http.MethodPost {
		session, ok := getSessionAfterRefresh()
		if !ok {
			http.Error(w, "no session found", http.StatusInternalServerError)
		}
		warehouseName := r.FormValue("warehouseName")
		warehousePosition := r.FormValue("warehousePosition")
		warehouseCapacity, err4 := strconv.Atoi(r.FormValue("warehouseCapacity"))
		if err4 != nil {
			http.Error(w, err4.Error(), http.StatusInternalServerError)
		}
		createWarehouse(w, r, session, warehouseName, warehousePosition, warehouseCapacity)
		return
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// createWarehouse creates a new warehouse in the /warehouses page
func createWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseName string, warehousePosition string, warehouseCapacity int) {
	err2 := authManager.CreateWarehouse(session.id, warehouseName, warehousePosition, warehouseCapacity)
	if err2 != nil {
		setFlashMessage(w, "error", err2.Error())
		http.Redirect(w, r, "/warehouses", http.StatusFound)
		return
	}
	warehouse, err3 := authManager.FindWarehouseByName(session.id, warehouseName)
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
		return
	}
	if len(warehouse) == 0 {
		http.Error(w, "Newly created warehouse not found", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/warehouse/"+strconv.Itoa(int(warehouse[0].ID)), http.StatusFound)
	return
}

// ConsumeItemHandler implements the consumption of items in a certain warehouse through the /item/{id:[0-9]+} page
func ConsumeItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, amount, itemID := collectData(w, r)
		warehouseID, err2 := strconv.Atoi(r.FormValue("warehouseID"))
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
			return
		}
		err := authManager.ConsumeItems(session.id, uint(itemID), uint(warehouseID), amount)
		if err != nil {
			setFlashMessage(w, "error", err.Error())
			http.Redirect(w, r, "/item/"+mux.Vars(r)["itemID"], http.StatusFound)
			return
		}
		http.Redirect(w, r, "/items", http.StatusFound)
		return
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// SupplyItemHandler similar to ConsumeItemHandler
func SupplyItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, amount, itemID := collectData(w, r)
		warehouseID, err2 := strconv.Atoi(r.FormValue("warehouseID"))
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
			return
		}
		err := authManager.SupplyItems(session.id, uint(itemID), uint(warehouseID), amount)
		if err != nil {
			setFlashMessage(w, "error", err.Error())
			http.Redirect(w, r, "/item/"+mux.Vars(r)["itemID"], http.StatusFound)
			return
		}
		http.Redirect(w, r, "/items", http.StatusFound)
		return
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// collectData is a utility function to collect data
func collectData(w http.ResponseWriter, r *http.Request) (rSession userSession, rAmount int, rItemID int) {
	session, ok := getSessionAfterRefresh()
	if !ok {
		http.Error(w, "no session found", http.StatusInternalServerError)
	}
	amount, err1 := strconv.Atoi(r.FormValue("amount"))
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}
	itemID, err3 := strconv.Atoi(mux.Vars(r)["itemID"])
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
		return
	}
	return session, amount, itemID
}

// TransferItemHandler transfers items from a warehouse to another. It's accessible from /item/{id}/transfer path
func TransferItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, amount, itemID := collectData(w, r)
		srcID, err1 := strconv.Atoi(r.FormValue("srcID"))
		if err1 != nil {
			http.Error(w, err1.Error(), http.StatusInternalServerError)
			return
		}
		destID, err2 := strconv.Atoi(r.FormValue("destID"))
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
			return
		}
		err3 := authManager.TransferItems(session.id, uint(itemID), uint(srcID), amount, uint(destID))
		if err3 != nil {
			setFlashMessage(w, "error", err3.Error())
			http.Redirect(w, r, "/item/"+mux.Vars(r)["itemID"], http.StatusFound)
			return
		}
		http.Redirect(w, r, "/items", http.StatusFound)
		return
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// ItemHandler handlers the operations on the /item/{id} page
func ItemHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := getSessionAfterRefresh()
	if !ok {
		http.Error(w, "no session found", http.StatusInternalServerError)
	}
	vars := mux.Vars(r)
	itemIDStr := vars["itemID"]
	itemID, err1 := strconv.Atoi(itemIDStr)
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		{
			getItem(w, r, session, itemID)
			return
		}
	case http.MethodPut:
		{
			putItem(w, r, session, itemID)
			return
		}
	case http.MethodDelete:
		{
			deleteItem(w, r, session, itemID)
			return
		}
	default:
		{
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

// deleteItem allows for the elimination of items from the /item page
func deleteItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int) {
	err2 := authManager.DeleteItem(session.id, uint(itemID))
	if err2 != nil {
		setFlashMessage(w, "error", err2.Error())
		http.Redirect(w, r, "/items", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/items", http.StatusFound)
	return
}

// putItem allows for the modification of items from the corresponding /item page
func putItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int) {
	itemName := r.FormValue("itemName")
	itemCategory := r.FormValue("itemCategory")
	itemDescription := r.FormValue("itemDescription")
	updateItem(w, r, session, itemID, itemName, itemCategory, itemDescription)
	return
}

// utility method extracted to increase code readability
func updateItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int, itemName string, itemCategory string, itemDescription string) {
	_, err2 := authManager.FindItemByID(session.id, uint(itemID))
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	err3 := authManager.UpdateItem(session.id, uint(itemID), itemName, itemCategory, itemDescription)
	if err3 != nil {
		setFlashMessage(w, "error", err3.Error())
		http.Redirect(w, r, "/item/"+strconv.Itoa(itemID), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/items", http.StatusFound)
	return
}

// getItem shows the corresponding /item/{id} page
func getItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int) {
	item, err2 := authManager.FindItemByID(session.id, uint(itemID))
	if err2 != nil {
		NotFoundHandler(w, r)
		return
	}
	page := fillPage(w, r, session, itemID, item)
	err3 := templates.ExecuteTemplate(w, "item.html", page)
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
	}
	return
}

// collects data and fills the page struct
func fillPage(w http.ResponseWriter, r *http.Request, session userSession, itemID int, item model.Item) ItemPage {
	hostingWarehouses, err1 := findHostingWarehouses(w, session, itemID)
	allWarehouses, err2 := authManager.ListAllWarehouses(session.id)
	var page ItemPage
	page.LoggedIn = true
	page.Item = item
	page.HostingWarehouses = hostingWarehouses
	page.AllWarehouses = allWarehouses
	page.LOError = processFlashMessage(w, r, "lOError")
	page.APPError = processFlashMessage(w, r, "error")
	if err2 != nil {
		page.APPError = err2.Error()
	}
	if err1 != nil {
		page.APPError = err1.Error()
	}
	return page
}

// utility method to find which warehouses hosts the given item
func findHostingWarehouses(w http.ResponseWriter, session userSession, itemID int) ([]model.Warehouse, error) {
	warehousePacks, err := authManager.FindWarehousesForItem(session.id, uint(itemID))
	hostWarehouses := make([]model.Warehouse, 0)
	for _, v := range warehousePacks {
		warehouse, err := authManager.FindWarehouseByID(session.id, v.WarehouseID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		hostWarehouses = append(hostWarehouses, warehouse)
	}
	return hostWarehouses, err
}

// handlers the Warehouse page
func WarehouseHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := getSessionAfterRefresh()
	if !ok {
		http.Error(w, "no session found", http.StatusInternalServerError)
	}
	vars := mux.Vars(r)
	warehouseIDStr := vars["warehouseID"]
	warehouseID, err1 := strconv.Atoi(warehouseIDStr)
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		{
			getWarehouse(w, r, session, warehouseID)
			return
		}
	case http.MethodPut:
		{
			putWarehouse(w, r, session, warehouseID)
			return
		}
	case http.MethodDelete:
		{
			deleteWarehouse(w, r, session, warehouseID)
			return
		}
	default:
		{
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// deletes a given warehouse by id
func deleteWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseID int) {
	err2 := authManager.DeleteWarehouse(session.id, uint(warehouseID))
	if err2 != nil {
		setFlashMessage(w, "error", err2.Error())
		http.Redirect(w, r, "/warehouses", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/warehouses", http.StatusFound)
	return
}

// updates the info about a warehouse given its id
func putWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseID int) {
	warehouseName := r.FormValue("warehouseName")
	warehousePosition := r.FormValue("warehousePosition")
	warehouseCapacity, err4 := strconv.Atoi(r.FormValue("warehouseCapacity"))
	if err4 != nil {
		http.Error(w, err4.Error(), http.StatusInternalServerError)
		return
	}
	updateWarehouse(w, r, session, warehouseID, warehouseName, warehousePosition, warehouseCapacity)
	return
}

// utility method extracted from putWarehouse
func updateWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseID int, warehouseName string, warehousePosition string, warehouseCapacity int) {
	_, err3 := authManager.FindWarehouseByID(session.id, uint(warehouseID))
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
		return
	}
	err5 := authManager.UpdateWarehouse(session.id, uint(warehouseID), warehouseName, warehousePosition, warehouseCapacity)
	if err5 != nil {
		setFlashMessage(w, "error", err5.Error())
		http.Redirect(w, r, "/warehouse/"+strconv.Itoa(warehouseID), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/warehouses", http.StatusFound)
	return
}

// getWarehouse shows the warehouse page
func getWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseID int) {
	warehouse, err2 := authManager.FindWarehouseByID(session.id, uint(warehouseID))
	if err2 != nil {
		NotFoundHandler(w, r)
		return
	}
	var page WarehousePage
	page.LoggedIn = true
	page.APPError = processFlashMessage(w, r, "error")
	page.LOError = processFlashMessage(w, r, "lOError")
	page.Warehouse = warehouse
	err3 := templates.ExecuteTemplate(w, "warehouse.html", page)
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// operates the search page
func ItemsSearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		{
			getItemSearchPage(w, r)
			return
		}
	case http.MethodPost:
		{
			searchItem(w, r)
			return
		}
	default:
		{
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// allows for the research of items by id name or keyword
func searchItem(w http.ResponseWriter, r *http.Request) {
	var page SearchPage
	page.LoggedIn = true
	session, ok := getSessionAfterRefresh()
	if !ok {
		http.Error(w, "no session found", http.StatusInternalServerError)
	}
	itemIDStr := r.FormValue("itemID")
	itemName := r.FormValue("itemName")
	itemCategory := r.FormValue("itemCategory")
	itemDescription := r.FormValue("itemDescription")
	if itemIDStr != "" {
		searchItemID(w, r, itemIDStr, session, page)
		return
	} else if itemName != "" {
		searchItemName(w, r, session, itemName, page)
		return
	} else if itemDescription != "" {
		searchItemKeyword(w, r, session, itemDescription, itemCategory, page)
		return
	} else if itemCategory != "" {
		searchItemCategory(w, r, session, itemCategory, page)
		return
	}
	setFlashMessage(w, "error", "Missing search arguments")
	http.Redirect(w, r, "/items_search", http.StatusFound)
	return
}

// searches item by category
func searchItemCategory(w http.ResponseWriter, r *http.Request, session userSession, itemCategory string, page SearchPage) {
	resItems, err1 := authManager.FindItemsByCategory(session.id, itemCategory)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/items_search", http.StatusFound)
		return
	}
	page.Items = resItems
	err := templates.ExecuteTemplate(w, "items_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// extracted method to improve readability
func searchItemKeyword(w http.ResponseWriter, r *http.Request, session userSession, itemDescription string, itemCategory string, page SearchPage) {
	resItems, err1 := authManager.FindItemsByKeyword(session.id, itemDescription)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/items_search", http.StatusFound)
		return
	}
	if itemCategory != "" {
		temp := make([]model.Item, 0)
		for _, item := range resItems {
			if item.Category == itemCategory {
				temp = append(temp, item)
			}
		}
		resItems = temp
	}
	page.Items = resItems
	err2 := templates.ExecuteTemplate(w, "items_search.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// like before
func searchItemName(w http.ResponseWriter, r *http.Request, session userSession, itemName string, page SearchPage) {
	resItem, err1 := authManager.FindItemByName(session.id, itemName)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/items_search", http.StatusFound)
		return
	}
	page.Items = resItem
	err2 := templates.ExecuteTemplate(w, "items_search.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// like before
func searchItemID(w http.ResponseWriter, r *http.Request, itemIDStr string, session userSession, page SearchPage) {
	itemID, err1 := strconv.Atoi(itemIDStr)
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}
	resItem, err2 := authManager.FindItemByID(session.id, uint(itemID))
	if err2 != nil {
		setFlashMessage(w, "error", err2.Error())
		http.Redirect(w, r, "/items_search", http.StatusFound)
		return
	}
	page.Items = append(page.Items, resItem)
	err := templates.ExecuteTemplate(w, "items_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// processes and presents the page for item searching
func getItemSearchPage(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.LoggedIn = true
	page.APPError = processFlashMessage(w, r, "error")
	page.LOError = processFlashMessage(w, r, "lOError")
	err := templates.ExecuteTemplate(w, "items_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// WarehousesSearchHandler manages the requests on the /warehouses/search path
func WarehousesSearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		{
			getWarehouseSearchPage(w, r)
			return
		}
	case http.MethodPost:
		{
			searchWarehouse(w, r)
			return
		}
	default:
		{
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func searchWarehouse(w http.ResponseWriter, r *http.Request) {
	var page SearchPage
	page.LoggedIn = true
	session, ok := getSessionAfterRefresh()
	if !ok {
		http.Error(w, "no session found", http.StatusInternalServerError)
	}
	warehouseIDStr := r.FormValue("warehouseID")
	warehouseName := r.FormValue("warehouseName")
	warehousePosition := r.FormValue("warehousePosition")
	if warehouseIDStr != "" {
		searchWarehouseID(w, r, warehouseIDStr, session, page)
		return
	} else if warehouseName != "" {
		searchWarehouseName(w, r, session, warehouseName, page)
		return
	} else if warehousePosition != "" {
		searchWarehousePosition(w, r, session, warehousePosition, page)
		return
	}
	setFlashMessage(w, "error", "Missing search arguments")
	http.Redirect(w, r, "/warehouses_search", http.StatusFound)
	return
}

func searchWarehousePosition(w http.ResponseWriter, r *http.Request, session userSession, warehousePosition string, page SearchPage) {
	resWarehouses, err1 := authManager.FindWarehousesByPosition(session.id, warehousePosition)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/warehouses_search", http.StatusFound)
		return
	}
	page.Warehouses = resWarehouses
	err2 := templates.ExecuteTemplate(w, "warehouses_search.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
	return
}

func searchWarehouseName(w http.ResponseWriter, r *http.Request, session userSession, warehouseName string, page SearchPage) {
	resWarehouse, err1 := authManager.FindWarehouseByName(session.id, warehouseName)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/warehouses_search", http.StatusFound)
		return
	}
	page.Warehouses = resWarehouse
	err2 := templates.ExecuteTemplate(w, "warehouses_search.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func searchWarehouseID(w http.ResponseWriter, r *http.Request, warehouseIDStr string, session userSession, page SearchPage) {
	warehouseID, err1 := strconv.Atoi(warehouseIDStr)
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}
	warehouse, err2 := authManager.FindWarehouseByID(session.id, uint(warehouseID))
	if err2 != nil {
		setFlashMessage(w, "error", err2.Error())
		http.Redirect(w, r, "/warehouses_search", http.StatusFound)
		return
	}
	page.Warehouses = append(page.Warehouses, warehouse)
	err := templates.ExecuteTemplate(w, "warehouses_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func getWarehouseSearchPage(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.LoggedIn = true
	page.APPError = processFlashMessage(w, r, "error")
	page.LOError = processFlashMessage(w, r, "lOError")
	err := templates.ExecuteTemplate(w, "warehouses_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// Implements the default behaviour of the wab APP when dealing with invalid URLs
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := templates.ExecuteTemplate(w, "not_found.html", r.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// utility method to quickly get the session_token of the client
func getSession(w http.ResponseWriter, r *http.Request) (userSession, bool) {
	c1, err1 := r.Cookie("session_token")
	if err1 != nil && !errors.Is(err1, http.ErrNoCookie) {
		http.Error(w, err1.Error(), http.StatusBadRequest)
	}
	session, ok := userSessions[c1.Value]
	return session, ok
}

// SessionIsPresentHandler wraps around the login and register pages to redirect requests by users who are already logged in
func SessionIsPresentHandler(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err == nil {
			session, ok := userSessions[c.Value]
			if ok && !session.isExpired() {
				setFlashMessage(w, "flash", "You already have an active session!")
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

// allows for the registration of users
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		{
			postRegister(w, r)
			return
		}
	case http.MethodGet:
		{
			getRegister(w, r)
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// manages GET requests to the /register resource
func getRegister(w http.ResponseWriter, r *http.Request) {
	message := processFlashMessage(w, r, "error")
	err := templates.ExecuteTemplate(w, "register.html", &Page{APPError: message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// manages the POST requests to the /register resource. Register users.
func postRegister(w http.ResponseWriter, r *http.Request) {
	newUsername := r.FormValue("newUsername")
	newPassword1 := r.FormValue("newPassword1")
	newPassword2 := r.FormValue("newPassword2")
	if newPassword1 != newPassword2 {
		setFlashMessage(w, "error", "The passwords don't match")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}
	err := authManager.Register(newUsername, newPassword1)
	if err != nil {
		setFlashMessage(w, "error", err.Error())
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
	return
}

// operates the /login page
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		{
			postLogin(w, r)
			return
		}
	case http.MethodGet:
		{
			getLogin(w, r)
			return
		}
	default:
		{
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

}

// manages the GET requests
func getLogin(w http.ResponseWriter, r *http.Request) {
	message := processFlashMessage(w, r, "error")
	err := templates.ExecuteTemplate(w, "login.html", &Page{APPError: message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// manages the POST requests by logging in the user.
func postLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	userID, err := authManager.Login(username, password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		setFlashMessage(w, "error", err.Error())
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	c, err2 := r.Cookie("session_token")
	var token string
	if errors.Is(err2, http.ErrNoCookie) {
		token = ""
	} else if err2 != nil {
		token = c.Value
	} else {
		http.Error(w, "unexpected error when accessing cookie", http.StatusInternalServerError)
		return
	}
	createNewSession(w, username, userID, token)
	http.Redirect(w, r, "/home", http.StatusFound)
	return
}

// logs the user out
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, done := checkForActiveSession(w, r)
		if done {
			return
		}
		err1 := authManager.Logout(session.alias)
		if err1 != nil {
			setFlashMessage(w, "lOError", err1.Error())
			referer := r.Header.Get("Referer")
			if referer == "" {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			http.Redirect(w, r, referer, http.StatusFound)
			return
		}
		c, _ := r.Cookie("session_token")
		delete(userSessions, c.Value)
		newC := http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		}
		http.SetCookie(w, &newC)
		http.Redirect(w, r, "/home", http.StatusFound)
		return
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func AccountHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		{
			getAccount(w, r)
			return
		}
	case http.MethodPost:
		{
			postAccount(w, r)
			return
		}
	default:
		{
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func getSessionAfterRefresh() (userSession, bool) {
	token := tempCookieContainer.Value
	session, ok := userSessions[token]
	return session, ok
}

func postAccount(w http.ResponseWriter, r *http.Request) {
	token := tempCookieContainer.Value
	username := userSessions[token].alias
	oldPassword := r.FormValue("oldPassword")
	newPassword := r.FormValue("newPassword")
	err := authManager.ChangePassword(username, oldPassword, newPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		setFlashMessage(w, "error", err.Error())
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/home", http.StatusFound)
	return
}

func getAccount(w http.ResponseWriter, r *http.Request) {
	var page AccountPage
	session, ok := getSessionAfterRefresh()
	if !ok {
		http.Error(w, "Session not found", http.StatusInternalServerError)
		return
	}
	page.LoggedIn = true
	page.LOError = processFlashMessage(w, r, "lOError")
	page.APPMsg = evaluateItems(session)
	page.AuthMsg = processFlashMessage(w, r, "flash")
	page.Username = session.alias
	err := templates.ExecuteTemplate(w, "account.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// capture cookies and eliminates them
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

// creates new cookies
func setFlashMessage(w http.ResponseWriter, name string, value string) {
	cookie := http.Cookie{
		Name:  name,
		Value: base64.URLEncoding.EncodeToString([]byte(value)),
	}
	http.SetCookie(w, &cookie)
}

// check if the user has an active session on the web APP and returns it.
func checkForActiveSession(w http.ResponseWriter, r *http.Request) (userSession, bool) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			http.Error(w, "No active session found", http.StatusBadRequest)
			return userSession{}, true
		} else {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return userSession{}, true
		}
	}
	session, ok := userSessions[c.Value]
	if !ok || session.isExpired() {
		http.Error(w, "Expired or wrong session found", http.StatusBadRequest)
		return userSession{}, true
	}
	return session, false
}

// deletes expired userSessions from the environment
func ManageUserSessions() {
	time.Sleep(30 * time.Second)
	expiredK := make([]string, 0)
	for k, v := range userSessions {
		if v.isExpired() {
			expiredK = append(expiredK, k)
		}
	}
	for _, k := range expiredK {
		err := authManager.Logout(userSessions[k].alias)
		if err != nil {
			panic(err)
		}
		delete(userSessions, k)
	}
}

// Runs the APP
func RunAPP() {
	router := BuildAPPRouter()
	srv := &http.Server{
		Handler:      router,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}
	go ManageUserSessions()
	log.Fatal(srv.ListenAndServe())
}

// uses gorilla mux to route the requests to the corresponding handler
func BuildAPPRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", SessionIsAbsentHomeHandler(HomeHandler))
	router.HandleFunc("/home", SessionIsAbsentHomeHandler(HomeHandler))
	router.HandleFunc("/login", SessionIsPresentHandler(LoginHandler))
	router.HandleFunc("/register", SessionIsPresentHandler(RegisterHandler))
	router.HandleFunc("/logout", LogoutHandler)
	router.HandleFunc("/account", SessionIsAbsentRedirectHandler(RefreshHandler(AccountHandler)))
	router.HandleFunc("/items", SessionIsAbsentRedirectHandler(RefreshHandler(ItemsHandler)))
	router.HandleFunc("/warehouses", SessionIsAbsentRedirectHandler(RefreshHandler(WarehousesHandler)))
	router.HandleFunc("/item/{itemID:[0-9]+}", SessionIsAbsentRedirectHandler(RefreshHandler(ItemHandler)))
	router.HandleFunc("/warehouse/{warehouseID:[0-9]+}", SessionIsAbsentRedirectHandler(RefreshHandler(WarehouseHandler)))
	router.HandleFunc("/items/search", SessionIsAbsentRedirectHandler(RefreshHandler(ItemsSearchHandler)))
	router.HandleFunc("/warehouses/search", SessionIsAbsentRedirectHandler(RefreshHandler(WarehousesSearchHandler)))
	router.HandleFunc("/item/{itemID:[0-9]+}/supply", SessionIsAbsentRedirectHandler(RefreshHandler(SupplyItemHandler)))
	router.HandleFunc("/item/{itemID:[0-9]+}/consume", SessionIsAbsentRedirectHandler(RefreshHandler(ConsumeItemHandler)))
	router.HandleFunc("/item/{itemID:[0-9]+}/transfer", SessionIsAbsentRedirectHandler(RefreshHandler(TransferItemHandler)))
	router.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
	return router
}
