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
	"regexp"
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
var authManager, _ = auth.LoadAuthManager(func(name string) (model.WarehouseRepository, error) {
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

type WarehousesPage struct {
	Page
	Content []model.Warehouse
}

type ItemsPage struct {
	Page
	Content []model.Item
}

type ItemPage struct {
	Page
	Item              model.Item
	HostingWarehouses []model.Warehouse
	AllWarehouses     []model.Warehouse
}

type WarehousePage struct {
	Page
	Warehouse model.Warehouse
}

type SearchPage struct {
	Page
	Warehouses []model.Warehouse
	Items      []model.Item
}

func SessionIsAbsentHomeHandler(nextHandler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				err1 := templates.ExecuteTemplate(w, "templates/home.html", &Page{LoggedIn: false, AuthMsg: "Login to begin using the app!"})
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
			err1 := templates.ExecuteTemplate(w, "templates/home.html", &Page{LoggedIn: false, AuthMsg: "Previous session expired! Login again.!"})
			if err1 != nil {
				http.Error(w, err1.Error(), http.StatusInternalServerError)
			}
			return
		}
		nextHandler(w, r)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session := accessSession(w, r)
	var page Page
	page.LoggedIn = true
	page.AuthMsg = processFlashMessage(w, r, "flash")
	page.LOError = processFlashMessage(w, r, "lOError")
	page.APPMsg = evaluateItems(session)
	err2 := templates.ExecuteTemplate(w, "templates/home.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
}

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

func SessionIsAbsentRedirectHandler(nextHandler http.Handler) http.HandlerFunc {
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

func ItemsHandler(w http.ResponseWriter, r *http.Request) {
	session := accessSession(w, r)
	var page ItemsPage
	page.LoggedIn = true
	page.LOError = processFlashMessage(w, r, "lOError")
	items, err2 := authManager.ListAllItems(session.id)
	if err2 != nil {
		page.APPError = err2.Error()
		err3 := templates.ExecuteTemplate(w, "templates/items.html", page)
		if err3 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
		}
		return
	}
	page.APPError = processFlashMessage(w, r, "error")
	page.Content = items
	page.APPMsg = evaluateItems(session)
	err4 := templates.ExecuteTemplate(w, "templates/items.html", page)
	if err4 != nil {
		http.Error(w, err4.Error(), http.StatusInternalServerError)
	}
	return
}

func WarehousesHandler(w http.ResponseWriter, r *http.Request) {
	var page WarehousesPage
	page.LoggedIn = true
	page.LOError = processFlashMessage(w, r, "lOError")
	session := accessSession(w, r)
	warehouses, err1 := authManager.ListAllWarehouses(session.id)
	if err1 != nil {
		page.APPError = err1.Error()
		err2 := templates.ExecuteTemplate(w, "templates/warehouses.html", page)
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
		}
		return
	}
	page.Content = warehouses
	err2 := templates.ExecuteTemplate(w, "templates/warehouses.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
	return
}

func ConsumeItemHandler(w http.ResponseWriter, r *http.Request) {
	session, amount, itemID := collectData(w, r)
	warehouseID, err2 := strconv.Atoi(r.FormValue("warehouseID"))
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
	err := authManager.ConsumeItems(session.id, uint(itemID), uint(warehouseID), amount)
	if err != nil {
		setFlashMessage(w, "error", err.Error())
		http.Redirect(w, r, "/item/"+mux.Vars(r)["itemID"], http.StatusFound)
		return
	}
	http.Redirect(w, r, "/items", http.StatusFound)
	return
}

func SupplyItemHandler(w http.ResponseWriter, r *http.Request) {
	session, amount, itemID := collectData(w, r)
	warehouseID, err2 := strconv.Atoi(r.FormValue("warehouseID"))
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
	err := authManager.SupplyItems(session.id, uint(itemID), uint(warehouseID), amount)
	if err != nil {
		setFlashMessage(w, "error", err.Error())
		http.Redirect(w, r, "/item/"+mux.Vars(r)["itemID"], http.StatusFound)
		return
	}
	http.Redirect(w, r, "/items", http.StatusFound)
	return
}

func collectData(w http.ResponseWriter, r *http.Request) (rSession userSession, rAmount int, rItemID int) {
	session := accessSession(w, r)
	amount, err1 := strconv.Atoi(r.FormValue("amount"))
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
	}
	itemID, err3 := strconv.Atoi(mux.Vars(r)["itemID"])
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
	}
	return session, amount, itemID
}

func TransferItemHandler(w http.ResponseWriter, r *http.Request) {
	session, amount, itemID := collectData(w, r)
	srcID, err1 := strconv.Atoi(r.FormValue("srcID"))
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
	}
	destID, err2 := strconv.Atoi(r.FormValue("destID"))
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
	err3 := authManager.TransferItems(session.id, uint(itemID), uint(srcID), amount, uint(destID))
	if err3 != nil {
		setFlashMessage(w, "error", err3.Error())
		http.Redirect(w, r, "/item/"+mux.Vars(r)["itemID"], http.StatusFound)
		return
	}
	http.Redirect(w, r, "/items", http.StatusFound)
}

func ItemHandler(w http.ResponseWriter, r *http.Request) {
	session := accessSession(w, r)
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

func putItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int) {
	itemName := r.FormValue("itemName")
	itemCategory := r.FormValue("itemCategory")
	itemDescription := r.FormValue("itemDescription")
	referer := r.Header.Get("Referer")
	match, err4 := regexp.Match("/item/[0-9]+", []byte(referer))
	if err4 != nil {
		http.Error(w, err4.Error(), http.StatusInternalServerError)
		return
	}
	if referer == "/items" {
		createItem(w, r, session, itemID, itemName, itemCategory, itemDescription)
		return
	} else if match {
		updateItem(w, r, session, itemID, itemName, itemCategory, itemDescription)
		return
	}
	http.Error(w, "Bad access", http.StatusForbidden)
	return
}

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

func createItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int, itemName string, itemCategory string, itemDescription string) {
	err3 := authManager.CreateItem(session.id, itemName, itemCategory, itemDescription)
	if err3 != nil {
		setFlashMessage(w, "error", err3.Error())
		http.Redirect(w, r, "/items", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/item/"+strconv.Itoa(itemID), http.StatusFound)
	return
}

func getItem(w http.ResponseWriter, r *http.Request, session userSession, itemID int) {
	item, err2 := authManager.FindItemByID(session.id, uint(itemID))
	if err2 != nil {
		NotFoundHandler(w, r)
		return
	}
	page := fillPage(w, r, session, itemID, item)
	err3 := templates.ExecuteTemplate(w, "templates/item.html", page)
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
	}
	return
}

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

func WarehouseHandler(w http.ResponseWriter, r *http.Request) {
	session := accessSession(w, r)
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

func putWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseID int) {
	warehouseName := r.FormValue("warehouseName")
	warehousePosition := r.FormValue("warehousePosition")
	warehouseCapacity, err4 := strconv.Atoi(r.FormValue("warehouseCapacity"))
	referer := r.Header.Get("Referer")
	match, err2 := regexp.Match("/warehouse/[0-9]+", []byte(referer))
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	if referer == "/warehouses" {
		createWarehouse(w, r, session, warehouseID, err4, warehouseName, warehousePosition, warehouseCapacity)
		return
	} else if match {
		updateWarehouse(w, r, session, warehouseID, warehouseName, warehousePosition, warehouseCapacity)
		return
	}
	http.Error(w, "Bad access", http.StatusForbidden)
	return
}

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

func createWarehouse(w http.ResponseWriter, r *http.Request, session userSession, warehouseID int, err4 error, warehouseName string, warehousePosition string, warehouseCapacity int) {
	if err4 != nil {
		http.Error(w, err4.Error(), http.StatusInternalServerError)
		return
	}
	err3 := authManager.CreateWarehouse(session.id, warehouseName, warehousePosition, warehouseCapacity)
	if err3 != nil {
		setFlashMessage(w, "error", err3.Error())
		http.Redirect(w, r, "/warehouses", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/warehouse/"+strconv.Itoa(warehouseID), http.StatusFound)
	return
}

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
	err3 := templates.ExecuteTemplate(w, "templates/warehouse.html", page)
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
		return
	}
	return
}

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

func searchItem(w http.ResponseWriter, r *http.Request) {
	var page SearchPage
	page.LoggedIn = true
	session := accessSession(w, r)
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

func searchItemCategory(w http.ResponseWriter, r *http.Request, session userSession, itemCategory string, page SearchPage) {
	resItems, err1 := authManager.FindItemsByCategory(session.id, itemCategory)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/items_search", http.StatusFound)
		return
	}
	page.Items = resItems
	err := templates.ExecuteTemplate(w, "templates/items_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

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
	err2 := templates.ExecuteTemplate(w, "templates/items_search.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func searchItemName(w http.ResponseWriter, r *http.Request, session userSession, itemName string, page SearchPage) {
	resItem, err1 := authManager.FindItemByName(session.id, itemName)
	if err1 != nil {
		setFlashMessage(w, "error", err1.Error())
		http.Redirect(w, r, "/items_search", http.StatusFound)
		return
	}
	page.Items = resItem
	err2 := templates.ExecuteTemplate(w, "templates/items_search.html", page)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}
	return
}

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
	err := templates.ExecuteTemplate(w, "templates/items_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func getItemSearchPage(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.LoggedIn = true
	page.APPError = processFlashMessage(w, r, "error")
	page.LOError = processFlashMessage(w, r, "lOError")
	err := templates.ExecuteTemplate(w, "templates/items_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

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
	session := accessSession(w, r)
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
	err2 := templates.ExecuteTemplate(w, "templates/warehouses_search.html", page)
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
	err2 := templates.ExecuteTemplate(w, "templates/warehouses_search.html", page)
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
	err := templates.ExecuteTemplate(w, "templates/warehouses_search.html", page)
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
	err := templates.ExecuteTemplate(w, "templates/warehouses_search.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := templates.ExecuteTemplate(w, "templates/notfound.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func accessSession(w http.ResponseWriter, r *http.Request) userSession {
	c1, err1 := r.Cookie("session_token")
	if err1 != nil && !errors.Is(err1, http.ErrNoCookie) {
		http.Error(w, err1.Error(), http.StatusBadRequest)
	}
	session, _ := userSessions[c1.Value]
	return session
}

func SessionIsPresentHandler(nextHandler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
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

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	message := processFlashMessage(w, r, "error")
	err := templates.ExecuteTemplate(w, "templates/register.html", &Page{APPError: message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	message := processFlashMessage(w, r, "error")
	err := templates.ExecuteTemplate(w, "templates/login.html", &Page{APPError: message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	newUsername := r.FormValue("newUsername")
	newPassword1 := r.FormValue("newPassword1")
	newPassword2 := r.FormValue("newPassword2")
	if newPassword1 != newPassword2 {
		setFlashMessage(w, "error", "The passwords don't match")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	} else {
		err := authManager.Register(newUsername, newPassword1)
		if err != nil {
			setFlashMessage(w, "error", err.Error())
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
	userID, err := authManager.Login(username, password)
	if err != nil {
		setFlashMessage(w, "error", err.Error())
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	createNewSession(w, username, userID)
	http.Redirect(w, r, "/home", http.StatusFound)
	return
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
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
	newC := http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	}
	http.SetCookie(w, &newC)
	http.Redirect(w, r, "/home", http.StatusFound)
	return
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

func setFlashMessage(w http.ResponseWriter, name string, value string) {
	cookie := http.Cookie{
		Name:  name,
		Value: base64.URLEncoding.EncodeToString([]byte(value)),
	}
	http.SetCookie(w, &cookie)
}

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
		http.Error(w, "No active session found", http.StatusBadRequest)
		return userSession{}, true
	}
	return session, false
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
		err := authManager.Logout(userSessions[k].alias)
		if err != nil {
			panic(err)
		}
		delete(userSessions, k)
	}
}
