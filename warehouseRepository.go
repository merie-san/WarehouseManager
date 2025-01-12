package main

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type Warehouse struct {
	ID        uint `gorm:"primaryKey;<-:create;autoIncrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"unique;not null"`
	Position  string         `gorm:"not null"`
	Capacity  int            `gorm:"not null"`
}

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

type WarehouseItem struct {
	ItemID      uint `gorm:"primaryKey"`
	WarehouseID uint `gorm:"primaryKey"`
	Quantity    int  `gorm:"not null;default:0"`
}

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

type WarehouseRepository interface {
	FindItemByID(itemID uint) (Item, error)
	FindWarehouseByID(warehouseID uint) (Warehouse, error)
	FindItemsByKeyword(keyword string) ([]Item, error)
	FindItemByName(name string) (Item, error)
	FindWarehouseByName(name string) (Warehouse, error)
	FindWarehousesByPosition(position string) ([]Warehouse, error)
	FindItemsByCategory(category string) ([]Item, error)
	FindItemsInWarehouse(warehouseID uint) ([]LoadedItemPack, error)
	FindWarehousesForItem(itemID uint) ([]LoadedItemPack, error)
	CreateItem(name string, category string, description string) error
	CreateWarehouse(name string, position string, capacity int) error
	UpdateItem(itemID uint, name string, category string, description string) error
	UpdateWarehouse(warehouseID uint, name string, position string, capacity int) error
	DeleteItem(itemID uint) error
	DeleteWarehouse(warehouseID uint) error
	SupplyItems(itemID uint, warehouseID uint, quantity int) error
	ConsumeItems(itemID uint, warehouseID uint, quantity int) error
	TransferItems(itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error
	Close() error
}

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
	err := r.DB.Where("name LIKE ?", "%"+keyword+"%").Find(&items).Error
	return items, err
}

func (r *GORMSQLiteWarehouseRepository) FindItemByName(name string) (Item, error) {
	var item Item
	err := r.DB.First(&item, "name = ?", name).Error
	return item, err
}

func (r *GORMSQLiteWarehouseRepository) FindWarehouseByName(name string) (Warehouse, error) {
	var warehouse Warehouse
	err := r.DB.First(&warehouse, "name = ?", name).Error
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
	var item Item
	var warehouse Warehouse
	var nItems int
	var warehouseItems []WarehouseItem
	err1 := r.DB.First(&item, itemID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.First(&warehouse, warehouseID).Error
	if err2 != nil {
		return err2
	}
	db := r.DB.Table("warehouse_items").Where("warehouse_id = ?", warehouseID).Find(&warehouseItems)
	err5 := db.Error
	if err5 != nil {
		return err5
	}
	if len(warehouseItems) != 0 {
		err6 := db.Select("SUM(quantity)").Scan(&nItems).Error
		if err6 != nil {
			return err6
		}
	} else {
		nItems = 0
	}
	if nItems+quantity > warehouse.Capacity {
		return errors.New("warehouse is full: " + strconv.Itoa(nItems+quantity) + " > " + strconv.Itoa(warehouse.Capacity))
	}
	err4 := r.DB.Model(&WarehouseItem{}).Where("item_id = ? AND warehouse_id = ?", itemID, warehouseID).Find(&warehouseItems).Error
	if err4 != nil {
		return err4
	}
	if len(warehouseItems) == 0 {
		err5 := r.DB.Create(&WarehouseItem{ItemID: itemID, WarehouseID: warehouseID, Quantity: quantity}).Error
		if err5 != nil {
			return err5
		}
	} else {
		warehouseItems[0].Quantity += quantity
		err6 := r.DB.Save(&warehouseItems[0]).Error
		if err6 != nil {
			return err6
		}
	}
	item.Quantity += quantity
	err3 := r.DB.Save(&item).Error
	if err3 != nil {
		return err3
	}
	return nil
}

func (r *GORMSQLiteWarehouseRepository) ConsumeItems(itemID uint, warehouseID uint, quantity int) error {
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
	err4 := r.DB.Model(&WarehouseItem{}).Where("item_id = ? AND warehouse_id = ?", itemID, warehouseID).Find(&warehouseItems).Error
	if err4 != nil {
		return err4
	}
	if len(warehouseItems) == 0 {
		return errors.New("item not found in specified warehouse")
	} else if warehouseItems[0].Quantity < quantity {
		return errors.New("not enough items in specified warehouse: " + strconv.Itoa(warehouseItems[0].Quantity) + " < " + strconv.Itoa(quantity))
	} else {
		warehouseItems[0].Quantity -= quantity
		err5 := r.DB.Save(&warehouseItems[0]).Error
		if err5 != nil {
			return err5
		}
	}
	item.Quantity -= quantity
	err3 := r.DB.Save(&item).Error
	if err3 != nil {
		return err3
	}
	return nil
}

func (r *GORMSQLiteWarehouseRepository) TransferItems(itemID uint, sourceWarehouseID uint, quantity int, destinationWarehouseID uint) error {
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
