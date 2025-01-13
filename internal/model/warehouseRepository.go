package model

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"strconv"
	"time"
)

// Warehouse is struct representing a model used to store information about registered warehouses for users
type Warehouse struct {
	ID        uint `gorm:"primaryKey;<-:create;autoIncrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"unique;not null"`
	Position  string         `gorm:"not null"`
	Capacity  int            `gorm:"not null"`
}

// Item is struct representing a model used to store information about registered items for users
type Item struct {
	ID          uint `gorm:"primaryKey;<-:create;autoIncrement"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"unique;not null"`
	Description string         `gorm:"default:'No description'"`
	Category    string         `gorm:"default:'No category'"`
	Quantity    int            `gorm:"not null;default:0"`
}

// WarehouseItem is a struct used to create a model with GORM representing the many-to-many association between Items and Warehouses
type WarehouseItem struct {
	ItemID      uint `gorm:"primaryKey"`
	WarehouseID uint `gorm:"primaryKey"`
	Quantity    int  `gorm:"not null;default:0"`
}

// LoadedItemPack is a struct representing a group of the same item in a certain warehouse
type LoadedItemPack struct {
	ItemID            uint
	ItemName          string
	ItemDescription   string
	ItemCategory      string
	ItemQuantity      int
	WarehouseID       uint
	WarehouseName     string
	WarehousePosition string
	WarehouseCapacity int
}

// WarehouseRepository is an interface used to define repositories used by the application
type WarehouseRepository interface {
	// FindItemByID searches for an item in the repository with the specified ID and return it as an Item struct
	FindItemByID(itemID uint) (Item, error)

	// FindWarehouseByID searches for a warehouse in the repository with the specified ID and return it as a Warehouse struct
	FindWarehouseByID(warehouseID uint) (Warehouse, error)

	// FindItemsByKeyword retrieves a list of items whose descriptions contain the given keyword. Returns an error if any occurs.
	FindItemsByKeyword(keyword string) ([]Item, error)

	// FindItemByName retrieves an Item from the repository based on its name.
	// It returns the Item in position 0 of the array or an empty slice if nothing is found.
	FindItemByName(name string) ([]Item, error)

	// FindWarehouseByName retrieves a Warehouse based on its name. Returns a slice of matches or an error on failure.
	FindWarehouseByName(name string) ([]Warehouse, error)

	// FindWarehousesByPosition retrieves a list of warehouses located at the specified position.
	// Returns a slice of Warehouse structs and an error if any issues occur during the query.
	FindWarehousesByPosition(position string) ([]Warehouse, error)

	// FindItemsByCategory retrieves a list of items that belong to the specified category from the repository.
	FindItemsByCategory(category string) ([]Item, error)

	// FindItemsInWarehouse retrieves a list of LoadedItemPack for a specific warehouse identified by warehouseID.
	// Returns an error if retrieval fails or the warehouseID is invalid.
	FindItemsInWarehouse(warehouseID uint) ([]LoadedItemPack, error)

	// FindWarehousesForItem retrieves a list of LoadedItemPack containing details of warehouses storing the given item.
	FindWarehousesForItem(itemID uint) ([]LoadedItemPack, error)

	// CreateItem creates a new item with the specified name, category, and description in the repository.
	CreateItem(name string, category string, description string) error

	// CreateWarehouse creates a new warehouse record with the specified name, position, and capacity.
	CreateWarehouse(name string, position string, capacity int) error

	// UpdateItem updates the details of an item identified by itemID, including its name, category, and description.
	UpdateItem(itemID uint, name string, category string, description string) error

	// UpdateWarehouse updates the warehouse information such as name, position, and capacity by its unique ID.
	UpdateWarehouse(warehouseID uint, name string, position string, capacity int) error

	// DeleteItem removes an item from the repository using its unique identifier when it is empty. Returns an error if the operation fails.
	DeleteItem(itemID uint) error

	// DeleteWarehouse removes a warehouse record from the system using its unique identifier (warehouseID) when it is empty. It returns an error if the deletion fails.
	DeleteWarehouse(warehouseID uint) error

	// SupplyItems adds the specified quantity of an item to the inventory of a given warehouse.
	// Returns an error if the item or warehouse is not found, if the warehouse is full, or if any database operation fails.
	SupplyItems(itemID uint, warehouseID uint, quantity int) error

	// ConsumeItems decreases the quantity of a specific item in a given warehouse by the specified amount. Returns an error if unsuccessful.
	ConsumeItems(itemID uint, warehouseID uint, quantity int) error

	// TransferItems transfers a specified quantity of an item from one warehouse to another. Returns an error on failure.
	TransferItems(itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error

	// Close closes the repository connection, releasing any allocated resources. Returns an error if the operation fails.
	Close() error
}

// GORMSQLiteWarehouseRepository offers an implementation of WarehouseRepository using GORM and SQLite
type GORMSQLiteWarehouseRepository struct {
	DB *gorm.DB
}

func NewGORMSQLiteWarehouseRepository(DBName string) (*GORMSQLiteWarehouseRepository, error) {
	database, err1 := gorm.Open(sqlite.Open(DBName), &gorm.Config{})
	if err1 != nil {
		return nil, err1
	}
	err2 := database.AutoMigrate(&Warehouse{}, &Item{}, &WarehouseItem{})
	if err2 != nil {
		return nil, err2
	}
	return &GORMSQLiteWarehouseRepository{DB: database}, nil
}

func (r *GORMSQLiteWarehouseRepository) Close() error {
	db, err := r.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (r *GORMSQLiteWarehouseRepository) CreateItem(name string, category string, description string) error {
	return r.DB.Create(&Item{Name: name, Category: category, Description: description}).Error
}

func (r *GORMSQLiteWarehouseRepository) CreateWarehouse(name string, position string, capacity int) error {
	return r.DB.Create(&Warehouse{Name: name, Position: position, Capacity: capacity}).Error
}

func (r *GORMSQLiteWarehouseRepository) UpdateItem(itemID uint, name string, category string, description string) error {
	var item Item
	err := r.DB.First(&item, itemID).Error
	if err != nil {
		return err
	}
	item.Name = name
	item.Description = description
	item.Category = category
	return r.DB.Save(&item).Error
}

func (r *GORMSQLiteWarehouseRepository) UpdateWarehouse(warehouseID uint, name string, position string, capacity int) error {
	var warehouse Warehouse
	err := r.DB.First(&warehouse, warehouseID).Error
	if err != nil {
		return err
	}
	warehouse.Name = name
	warehouse.Capacity = capacity
	warehouse.Position = position
	return r.DB.Save(&warehouse).Error
}

func (r *GORMSQLiteWarehouseRepository) FindItemByID(itemID uint) (Item, error) {
	var item Item
	err := r.DB.First(&item, itemID).Error
	return item, err
}

func (r *GORMSQLiteWarehouseRepository) FindWarehouseByID(warehouseID uint) (Warehouse, error) {
	var warehouse Warehouse
	err := r.DB.First(&warehouse, warehouseID).Error
	return warehouse, err
}

func (r *GORMSQLiteWarehouseRepository) FindItemsByKeyword(keyword string) ([]Item, error) {
	var items []Item
	err := r.DB.Where("description LIKE ?", "%"+keyword+"%").Find(&items).Error
	return items, err
}

func (r *GORMSQLiteWarehouseRepository) FindItemByName(name string) ([]Item, error) {
	var item []Item
	err := r.DB.Find(&item, "name = ?", name).Error
	return item, err
}

func (r *GORMSQLiteWarehouseRepository) FindWarehouseByName(name string) ([]Warehouse, error) {
	var warehouse []Warehouse
	err := r.DB.Find(&warehouse, "name = ?", name).Error
	return warehouse, err
}

func (r *GORMSQLiteWarehouseRepository) FindWarehousesByPosition(position string) ([]Warehouse, error) {
	var warehouses []Warehouse
	err := r.DB.Where("position = ?", position).Find(&warehouses).Error
	return warehouses, err
}

func (r *GORMSQLiteWarehouseRepository) FindItemsByCategory(category string) ([]Item, error) {
	var items []Item
	err := r.DB.Where("category = ?", category).Find(&items).Error
	return items, err
}

func (r *GORMSQLiteWarehouseRepository) FindItemsInWarehouse(warehouseID uint) ([]LoadedItemPack, error) {
	var correspondence []WarehouseItem
	var res []LoadedItemPack
	err1 := r.DB.Model(&WarehouseItem{}).Where("warehouse_id = ?", warehouseID).Find(&correspondence).Error
	if err1 != nil {
		return nil, err1
	}
	for _, v := range correspondence {
		temp1, err2 := r.FindItemByID(v.ItemID)
		temp2, err3 := r.FindWarehouseByID(v.WarehouseID)
		if err2 != nil {
			return nil, err2
		}
		if err3 != nil {
			return nil, err3
		}
		res = append(res, LoadedItemPack{
			ItemID:            v.ItemID,
			ItemName:          temp1.Name,
			ItemDescription:   temp1.Description,
			ItemCategory:      temp1.Category,
			ItemQuantity:      v.Quantity,
			WarehouseID:       warehouseID,
			WarehouseName:     temp2.Name,
			WarehousePosition: temp2.Position,
			WarehouseCapacity: temp2.Capacity,
		})
	}
	return res, nil
}

func (r *GORMSQLiteWarehouseRepository) FindWarehousesForItem(itemID uint) ([]LoadedItemPack, error) {
	var correspondence []WarehouseItem
	var res []LoadedItemPack
	err1 := r.DB.Model(&WarehouseItem{}).Where("item_id = ?", itemID).Find(&correspondence).Error
	if err1 != nil {
		return nil, err1
	}
	for _, v := range correspondence {
		temp1, err2 := r.FindWarehouseByID(v.WarehouseID)
		temp2, err3 := r.FindItemByID(v.ItemID)
		if err2 != nil {
			return nil, err2
		}
		if err3 != nil {
			return nil, err3
		}
		res = append(res, LoadedItemPack{
			ItemID:            v.ItemID,
			ItemName:          temp2.Name,
			ItemDescription:   temp2.Description,
			ItemCategory:      temp2.Category,
			ItemQuantity:      v.Quantity,
			WarehouseID:       v.WarehouseID,
			WarehouseName:     temp1.Name,
			WarehousePosition: temp1.Position,
			WarehouseCapacity: temp1.Capacity,
		})
	}
	return res, nil
}
func (r *GORMSQLiteWarehouseRepository) DeleteItem(itemID uint) error {
	var item Item
	err := r.DB.First(&item, itemID).Error
	if err != nil {
		return err
	}
	if item.Quantity > 0 {
		return errors.New("item is not empty")
	} else {
		return r.DB.Delete(&item).Error
	}
}

func (r *GORMSQLiteWarehouseRepository) DeleteWarehouse(warehouseID uint) error {
	var warehouse Warehouse
	var correspondence []WarehouseItem
	err1 := r.DB.First(&warehouse, warehouseID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.Model(&WarehouseItem{}).Where("warehouse_id = ?", warehouseID).Find(&correspondence).Error
	if err2 != nil {
		return err2
	}
	if len(correspondence) != 0 {
		return errors.New("warehouse is not empty")
	} else {
		return r.DB.Delete(&warehouse).Error
	}
}

func (r *GORMSQLiteWarehouseRepository) SupplyItems(itemID uint, warehouseID uint, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	var item Item
	var warehouse Warehouse
	var nItems int
	var warehouseItems1 []WarehouseItem
	var warehouseItems2 []WarehouseItem
	err1 := r.DB.First(&item, itemID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.First(&warehouse, warehouseID).Error
	if err2 != nil {
		return err2
	}
	db := r.DB.Table("warehouse_items").Where("warehouse_id = ?", warehouseID).Find(&warehouseItems1)
	err3 := db.Error
	if err3 != nil {
		return err3
	}
	if len(warehouseItems1) != 0 {
		err4 := db.Select("SUM(quantity)").Scan(&nItems).Error
		if err4 != nil {
			return err4
		}
	} else {
		nItems = 0
	}
	if nItems+quantity > warehouse.Capacity {
		return errors.New("warehouse is full: " + strconv.Itoa(nItems+quantity) + " > " + strconv.Itoa(warehouse.Capacity))
	}
	err5 := r.DB.Model(&WarehouseItem{}).Where("item_id = ? AND warehouse_id = ?", itemID, warehouseID).Find(&warehouseItems2).Error
	if err5 != nil {
		return err5
	}
	if len(warehouseItems2) == 0 {
		err6 := r.DB.Create(&WarehouseItem{ItemID: itemID, WarehouseID: warehouseID, Quantity: quantity}).Error
		if err6 != nil {
			return err6
		}
	} else {
		warehouseItems2[0].Quantity += quantity
		err7 := r.DB.Save(&warehouseItems2[0]).Error
		if err7 != nil {
			return err7
		}
	}
	item.Quantity += quantity
	err8 := r.DB.Save(&item).Error
	if err8 != nil {
		return err8
	}
	return nil
}

func (r *GORMSQLiteWarehouseRepository) ConsumeItems(itemID uint, warehouseID uint, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	var item Item
	var warehouse Warehouse
	var warehouseItems []WarehouseItem
	err1 := r.DB.First(&item, itemID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.First(&warehouse, warehouseID).Error
	if err2 != nil {
		return err2
	}
	if item.Quantity < quantity {
		return errors.New("not enough items: " + strconv.Itoa(item.Quantity) + " < " + strconv.Itoa(quantity))
	}
	err3 := r.DB.Model(&WarehouseItem{}).Where("item_id = ? AND warehouse_id = ?", itemID, warehouseID).Find(&warehouseItems).Error
	if err3 != nil {
		return err3
	}
	if len(warehouseItems) == 0 {
		return errors.New("item not found in specified warehouse")
	} else if warehouseItems[0].Quantity < quantity {
		return errors.New("not enough items in specified warehouse: " + strconv.Itoa(warehouseItems[0].Quantity) + " < " + strconv.Itoa(quantity))
	} else {
		warehouseItems[0].Quantity -= quantity
		err4 := r.DB.Save(&warehouseItems[0]).Error
		if err4 != nil {
			return err4
		}
	}
	item.Quantity -= quantity
	err5 := r.DB.Save(&item).Error
	if err5 != nil {
		return err5
	}
	return nil
}

func (r *GORMSQLiteWarehouseRepository) TransferItems(itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error {
	if quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	err1 := r.ConsumeItems(itemID, sourceWarehouseID, quantity)
	if err1 != nil {
		return err1
	}
	err2 := r.SupplyItems(itemID, destinationWarehouseID, quantity)
	if err2 != nil {
		return err2
	}
	return nil
}
