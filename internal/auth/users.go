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

// GeneralUserSessionManager is an interface managing user sessions and resource operations for items and warehouses.
type GeneralUserSessionManager interface {

	/* Authentication operations */

	// Save writes the session's data to permanent memory
	Save() error

	// Login initiates a new user session with the specified user instance
	Login(username string, password string) error

	// Logout terminates the current user session and clears any associated session data.
	Logout() error

	// Register registers a new user using the provided username and password, returning an error if the operation fails.
	Register(username string, password string) error

	// ChangePassword updates the user's password by validating the old password and setting a new one. Returns an error if failed.
	ChangePassword(oldPassword string, newPassword string) error

	/* Repository operations */

	FindItemByID(itemID uint) (model.Item, error)
	FindWarehouseByID(warehouseID uint) (model.Warehouse, error)
	FindItemsByKeyword(keyword string) ([]model.Item, error)
	FindItemByName(name string) ([]model.Item, error)
	FindWarehouseByName(name string) ([]model.Warehouse, error)
	FindWarehousesByPosition(position string) ([]model.Warehouse, error)
	FindItemsByCategory(category string) ([]model.Item, error)
	FindItemsInWarehouse(warehouseID uint) ([]model.LoadedItemPack, error)
	FindWarehousesForItem(itemID uint) ([]model.LoadedItemPack, error)
	CreateItem(name string, category string, description string) error
	CreateWarehouse(name string, position string, capacity int) error
	UpdateItem(itemID uint, name string, category string, description string) error
	UpdateWarehouse(warehouseID uint, name string, position string, capacity int) error
	DeleteItem(itemID uint) error
	DeleteWarehouse(warehouseID uint) error
	SupplyItems(itemID uint, warehouseID uint, quantity int) error
	ConsumeItems(itemID uint, warehouseID uint, quantity int) error
	TransferItems(itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error
}

// UserSessionManager implements GeneralUserSessionManager by using an underlying WarehouseRepository.
// The struct manages user sessions by matching requests to the correct database and
// updating a list of registered users which is timely synchronized with a json file
type UserSessionManager struct {
	Users       []User
	injector    func(string) (model.WarehouseRepository, error)
	DB          model.WarehouseRepository
	CurrentUser *User
}

// Singleton of UserSessionManager
var manager *UserSessionManager

// LoadUserManager initializes and returns a singleton UserSessionManager with user data loaded from a JSON file.
// It accepts a dependency injector function for providing WarehouseRepository implementations.
// Returns an error if user data decoding fails or if other file operations encounter issues.
func LoadUserManager(injector func(string) (model.WarehouseRepository, error)) (*UserSessionManager, error) {
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
		manager = &UserSessionManager{Users: users, injector: injector}
	}
	return manager, nil
}

func (manager *UserSessionManager) Save() error {
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

func (manager *UserSessionManager) Login(username string, password string) error {
	if manager.CurrentUser != nil {
		return errors.New("active session found, logout first")
	}
	for _, v := range manager.Users {
		if v.Username == username && v.EncryptedPassword == ShaHashing(password) {
			manager.CurrentUser = &v
			var err error
			manager.DB, err = manager.injector(v.AssignedDatabase)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("invalid username or password")
}

func (manager *UserSessionManager) Logout() error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	_ = manager.DB.Close()
	manager.CurrentUser = nil
	manager.DB = nil
	return nil
}

func ShaHashing(input string) string {
	plainText := []byte(input)
	sha256Hash := sha256.Sum256(plainText)
	return hex.EncodeToString(sha256Hash[:])
}

func (manager *UserSessionManager) Register(username string, password string) error {
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

func (manager *UserSessionManager) DeleteAllUsers() error {
	manager.Users = []User{}
	manager.CurrentUser = nil
	manager.DB = nil
	return manager.Save()
}

func (manager *UserSessionManager) ChangePassword(oldPassword string, newPassword string) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	if manager.CurrentUser.EncryptedPassword != ShaHashing(oldPassword) {
		return errors.New("incorrect password")
	}
	manager.CurrentUser.EncryptedPassword = ShaHashing(newPassword)
	manager.Users[manager.CurrentUser.UserID] = *manager.CurrentUser
	return manager.Save()
}

func (manager *UserSessionManager) FindItemByID(itemID uint) (model.Item, error) {
	if manager.CurrentUser == nil {
		return model.Item{}, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindItemByID(itemID)
}

func (manager *UserSessionManager) FindWarehouseByID(warehouseID uint) (model.Warehouse, error) {
	if manager.CurrentUser == nil {
		return model.Warehouse{}, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindWarehouseByID(warehouseID)
}

func (manager *UserSessionManager) FindItemsByKeyword(keyword string) ([]model.Item, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindItemsByKeyword(keyword)
}

func (manager *UserSessionManager) FindItemByName(name string) ([]model.Item, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindItemByName(name)
}

func (manager *UserSessionManager) FindWarehouseByName(name string) ([]model.Warehouse, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindWarehouseByName(name)
}

func (manager *UserSessionManager) FindWarehousesByPosition(position string) ([]model.Warehouse, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindWarehousesByPosition(position)
}

func (manager *UserSessionManager) FindItemsByCategory(category string) ([]model.Item, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindItemsByCategory(category)
}

func (manager *UserSessionManager) FindItemsInWarehouse(warehouseID uint) ([]model.LoadedItemPack, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindItemsInWarehouse(warehouseID)
}

func (manager *UserSessionManager) FindWarehousesForItem(itemID uint) ([]model.LoadedItemPack, error) {
	if manager.CurrentUser == nil {
		return nil, errors.New("user hasn't logged in yet")
	}
	return manager.DB.FindWarehousesForItem(itemID)
}

func (manager *UserSessionManager) CreateItem(name string, category string, description string) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.CreateItem(name, category, description)
}

func (manager *UserSessionManager) CreateWarehouse(name string, position string, capacity int) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.CreateWarehouse(name, position, capacity)
}

func (manager *UserSessionManager) UpdateItem(itemID uint, name string, category string, description string) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.UpdateItem(itemID, name, category, description)
}

func (manager *UserSessionManager) UpdateWarehouse(warehouseID uint, name string, position string, capacity int) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.UpdateWarehouse(warehouseID, name, position, capacity)
}

func (manager *UserSessionManager) DeleteItem(itemID uint) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.DeleteItem(itemID)
}

func (manager *UserSessionManager) DeleteWarehouse(warehouseID uint) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.DeleteWarehouse(warehouseID)
}

func (manager *UserSessionManager) SupplyItems(itemID uint, warehouseID uint, quantity int) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.SupplyItems(itemID, warehouseID, quantity)
}

func (manager *UserSessionManager) ConsumeItems(itemID uint, warehouseID uint, quantity int) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.ConsumeItems(itemID, warehouseID, quantity)
}

func (manager *UserSessionManager) TransferItems(itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error {
	if manager.CurrentUser == nil {
		return errors.New("user hasn't logged in yet")
	}
	return manager.DB.TransferItems(itemID, sourceWarehouseID, quantity, destinationWarehouseID)
}
