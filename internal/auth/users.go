package auth

import (
	"WarehouseManager/internal/model"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
)

// User represents an app user with unique ID, username, encrypted password, and assigned database.
type User struct {
	UserID            uint   `json:"userID"`
	Username          string `json:"username"`
	EncryptedPassword string `json:"password"`
	AssignedDatabase  string `json:"assignedDatabase"`
}

// GeneralAuthManager is an interface managing user sessions and resource operations for items and warehouses.
type GeneralAuthManager interface {

	/* Authentication operations */

	// Save writes the session's data to permanent memory
	Save() error

	// Login initiates a new user session with the specified user instance
	Login(username string, password string) (uint, error)

	// Logout terminates the current user session and clears any associated session data.
	Logout(username string) error

	// Register registers a new user using the provided username and password, returning an error if the operation fails.
	Register(username string, password string) error

	// ChangePassword updates the user's password by validating the old password and setting a new one. Returns an error if failed.
	ChangePassword(username string, oldPassword string, newPassword string) error

	// IsLoggedIn checks and returns true if a user is currently logged in, otherwise returns false.
	IsLoggedIn(username string) bool

	/* Repository operations */

	FindItemByID(userID uint, itemID uint) (model.Item, error)
	FindWarehouseByID(userID uint, warehouseID uint) (model.Warehouse, error)
	FindItemsByKeyword(userID uint, keyword string) ([]model.Item, error)
	FindItemByName(userID uint, name string) ([]model.Item, error)
	FindWarehouseByName(userID uint, name string) ([]model.Warehouse, error)
	FindWarehousesByPosition(userID uint, position string) ([]model.Warehouse, error)
	FindItemsByCategory(userID uint, category string) ([]model.Item, error)
	FindItemsInWarehouse(userID uint, warehouseID uint) ([]model.LoadedItemPack, error)
	FindWarehousesForItem(userID uint, itemID uint) ([]model.LoadedItemPack, error)
	CreateItem(userID uint, name string, category string, description string) error
	CreateWarehouse(userID uint, name string, position string, capacity int) error
	UpdateItem(userID uint, itemID uint, name string, category string, description string) error
	UpdateWarehouse(userID uint, warehouseID uint, name string, position string, capacity int) error
	DeleteItem(userID uint, itemID uint) error
	DeleteWarehouse(userID uint, warehouseID uint) error
	SupplyItems(userID uint, itemID uint, warehouseID uint, quantity int) error
	ConsumeItems(userID uint, itemID uint, warehouseID uint, quantity int) error
	TransferItems(userID uint, itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error
	ListAllItems(userID uint) ([]model.Item, error)
	ListAllWarehouses(userID uint) ([]model.Warehouse, error)
}

// AuthManager implements GeneralAuthManager by using an underlying WarehouseRepository.
// The struct manages user sessions by matching requests to the correct database and
// updating a list of registered users which is timely synchronized with a json file
type AuthManager struct {
	Users       []User
	injector    func(string) (model.WarehouseRepository, error)
	ActiveUsers []ActiveUser
}

type ActiveUser struct {
	User *User
	DB   model.WarehouseRepository
}

// Singleton of AuthManager
var manager *AuthManager

// LoadUserManager initializes and returns a singleton AuthManager with user data loaded from a JSON file.
// It accepts a dependency injector function for providing WarehouseRepository implementations.
// Returns an error if user data decoding fails or if other file operations encounter issues.
func LoadUserManager(injector func(string) (model.WarehouseRepository, error)) (*AuthManager, error) {
	if manager == nil {
		var f *os.File
		users := make([]User, 0)
		_, err1 := os.Stat("users.json")
		if err1 != nil {
			f, _ = os.Create("users.json")
			defer f.Close()
			encoder := json.NewEncoder(f)
			err2 := encoder.Encode(users)
			if err2 != nil {
				return nil, err2
			}
		} else {
			f, _ = os.Open("users.json")
			defer f.Close()
			decoder := json.NewDecoder(f)
			err2 := decoder.Decode(&users)
			if err2 != nil {
				return nil, err2
			}
		}
		manager = &AuthManager{Users: users, injector: injector}
	}
	return manager, nil
}

func (manager *AuthManager) Save() error {
	f, err := os.Create("users.json")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	err = encoder.Encode(manager.Users)
	defer f.Close()
	if err != nil {
		return err
	}
	return nil
}

func (manager *AuthManager) IsLoggedIn(username string) bool {
	found := false
	for _, activeUser := range manager.ActiveUsers {
		if activeUser.User.Username == username {
			found = true
		}
	}
	return found
}

func (manager *AuthManager) Login(username string, password string) (uint, error) {
	found := false
	for _, v := range manager.ActiveUsers {
		if v.User.Username == username {
			found = true
		}
	}
	if found {
		return 0, errors.New("user's active session found, log out first")
	}
	for _, v := range manager.Users {
		if v.Username == username && v.EncryptedPassword == ShaHashing(password) {
			db, err := manager.injector(v.AssignedDatabase)
			manager.ActiveUsers = append(manager.ActiveUsers, ActiveUser{User: &v, DB: db})
			if err != nil {
				return 0, err
			}
			return v.UserID, nil
		}
	}
	return 0, errors.New("invalid username or password")
}

func (manager *AuthManager) Logout(username string) error {
	found := false
	var index int
	for i, v := range manager.ActiveUsers {
		if v.User.Username == username {
			index = i
			found = true
		}
	}
	if !found {
		return errors.New("user hasn't logged in yet")
	}
	_ = manager.ActiveUsers[index].DB.Close()
	temp := make([]ActiveUser, 0)
	temp = append(temp, manager.ActiveUsers[:index]...)
	temp = append(temp, manager.ActiveUsers[index+1:]...)
	manager.ActiveUsers = temp
	return nil
}

func ShaHashing(input string) string {
	plainText := []byte(input)
	sha256Hash := sha256.Sum256(plainText)
	return hex.EncodeToString(sha256Hash[:])
}

func (manager *AuthManager) Register(username string, password string) error {
	for _, v := range manager.Users {
		if v.Username == username {
			return errors.New("username already exists")
		}
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	manager.Users = append(
		manager.Users,
		User{Username: username,
			EncryptedPassword: ShaHashing(password),
			UserID:            uint(len(manager.Users)),
			AssignedDatabase:  "usr" + strconv.Itoa(len(manager.Users)) + ".db"})
	return manager.Save()
}

func (manager *AuthManager) DeleteAllUsers() error {
	manager.Users = []User{}
	for _, v := range manager.ActiveUsers {
		_ = v.DB.Close()
	}
	manager.ActiveUsers = nil
	return manager.Save()
}

func (manager *AuthManager) ChangePassword(username string, oldPassword string, newPassword string) error {
	var index int
	found := false
	for i, v := range manager.ActiveUsers {
		if v.User.EncryptedPassword == ShaHashing(oldPassword) && v.User.Username == username {
			index = i
			found = true
		}
	}
	if !found {
		return errors.New("invalid username or old password")
	}
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters long")
	}
	manager.ActiveUsers[index].User.EncryptedPassword = ShaHashing(newPassword)
	manager.Users[manager.ActiveUsers[index].User.UserID] = *manager.ActiveUsers[index].User
	return manager.Save()
}

func (manager *AuthManager) FindItemByID(userID uint, itemID uint) (model.Item, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return model.Item{}, err
	}
	return manager.ActiveUsers[index].DB.FindItemByID(itemID)
}

func (manager *AuthManager) FindWarehouseByID(userID uint, warehouseID uint) (model.Warehouse, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return model.Warehouse{}, err
	}
	return manager.ActiveUsers[index].DB.FindWarehouseByID(warehouseID)
}

func (manager *AuthManager) FindItemsByKeyword(userID uint, keyword string) ([]model.Item, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindItemsByKeyword(keyword)
}

func (manager *AuthManager) FindItemByName(userID uint, name string) ([]model.Item, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindItemByName(name)
}

func (manager *AuthManager) FindWarehouseByName(userID uint, name string) ([]model.Warehouse, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindWarehouseByName(name)
}

func (manager *AuthManager) FindWarehousesByPosition(userID uint, position string) ([]model.Warehouse, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindWarehousesByPosition(position)
}

func (manager *AuthManager) FindItemsByCategory(userID uint, category string) ([]model.Item, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindItemsByCategory(category)
}

func (manager *AuthManager) FindItemsInWarehouse(userID uint, warehouseID uint) ([]model.LoadedItemPack, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindItemsInWarehouse(warehouseID)
}

func (manager *AuthManager) FindWarehousesForItem(userID uint, itemID uint) ([]model.LoadedItemPack, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.FindWarehousesForItem(itemID)
}

func (manager *AuthManager) CreateItem(userID uint, name string, category string, description string) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.CreateItem(name, category, description)
}

func (manager *AuthManager) CreateWarehouse(userID uint, name string, position string, capacity int) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.CreateWarehouse(name, position, capacity)
}

func (manager *AuthManager) UpdateItem(userID uint, itemID uint, name string, category string, description string) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.UpdateItem(itemID, name, category, description)
}

func (manager *AuthManager) UpdateWarehouse(userID uint, warehouseID uint, name string, position string, capacity int) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.UpdateWarehouse(warehouseID, name, position, capacity)
}

func (manager *AuthManager) DeleteItem(userID uint, itemID uint) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.DeleteItem(itemID)
}

func (manager *AuthManager) DeleteWarehouse(userID uint, warehouseID uint) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.DeleteWarehouse(warehouseID)
}

func (manager *AuthManager) SupplyItems(userID uint, itemID uint, warehouseID uint, quantity int) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.SupplyItems(itemID, warehouseID, quantity)
}

func (manager *AuthManager) ConsumeItems(userID uint, itemID uint, warehouseID uint, quantity int) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.ConsumeItems(itemID, warehouseID, quantity)
}

func (manager *AuthManager) TransferItems(userID uint, itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return err
	}
	return manager.ActiveUsers[index].DB.TransferItems(itemID, sourceWarehouseID, quantity, destinationWarehouseID)
}

func (manager *AuthManager) checkLogin(userID uint) (int, error) {
	found := false
	var index int
	for i, v := range manager.ActiveUsers {
		if v.User.UserID == userID {
			index = i
			found = true
		}
	}
	if !found {
		return 0, errors.New("user isn't logged in")
	}
	return index, nil
}

func (manager *AuthManager) ListAllItems(userID uint) ([]model.Item, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.ListAllItems()
}

func (manager *AuthManager) ListAllWarehouses(userID uint) ([]model.Warehouse, error) {
	index, err := manager.checkLogin(userID)
	if err != nil {
		return nil, err
	}
	return manager.ActiveUsers[index].DB.ListAllWarehouses()
}
