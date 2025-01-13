package auth

import (
	"WarehouseManager/internal/model"
	"encoding/json"
	"os"
)

type User struct {
	UserID            uint   `json:"userID"`
	Username          string `json:"username"`
	EncryptedPassword string `json:"password"`
	AssignedDatabase  string `json:"assignedDatabase"`
}

type GeneralUserSessionManager interface {
	Save() error
	Login(username string, password string) error
	Logout() error
	Register(username string, password string) error
	ChangePassword(oldPassword string, newPassword string) error

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

type UserSessionManager struct {
	Users       []User
	DB          *model.WarehouseRepository
	CurrentUser *User
}

var manager *UserSessionManager

func LoadUserManager() (*UserSessionManager, error) {
	if manager == nil {
		var f *os.File
		_, err1 := os.Stat("users.json")
		if err1 != nil {
			f, _ = os.Create("users.json")
		} else {
			f, _ = os.Open("users.json")
		}
		decoder := json.NewDecoder(f)
		var users []User
		err2 := decoder.Decode(&users)
		if err2 != nil {
			return nil, err2
		}
		manager = &UserSessionManager{Users: users}
		_ = f.Close()
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
	if err != nil {
		return err
	}
	return nil
}
