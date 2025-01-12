package main

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"time"
)

type Warehouse struct {
	ID        uint `gorm:"primaryKey;<-:create;autoIncrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"unique;not null"`
	Position  string         `gorm:"not null"`
	Capacity  uint           `gorm:"not null"`
}

type Item struct {
	ID          uint `gorm:"primaryKey;<-:create;autoIncrement"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"unique;not null"`
	Description string         `gorm:"default:'No description'"`
	Category    string         `gorm:"default:'No category'"`
	Quantity    uint           `gorm:"not null;default:0"`
}

type WarehouseItems struct {
	ItemID      uint `gorm:"primaryKey"`
	WarehouseID uint `gorm:"primaryKey"`
	Quantity    uint `gorm:"not null;default:0"`
}

type LoadedItemPack struct {
	ItemID            uint
	ItemName          string
	ItemDescription   string
	ItemCategory      string
	ItemQuantity      uint
	WarehouseID       uint
	WarehouseName     string
	WarehousePosition string
	WarehouseCapacity uint
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
	CreateWarehouse(name string, position string, capacity uint) error
	UpdateItem(itemID uint, name string, category string, description string) error
	UpdateWarehouse(warehouseID uint, name string, position string, capacity uint) error
	DeleteItem(itemID uint) error
	DeleteWarehouse(warehouseID uint) error
	SupplyItems(itemID uint, warehouseID uint, quantity uint) error
	ConsumeItems(itemID uint, warehouseID uint, quantity uint) error
	TransferItems(itemID uint, sourceWarehouseID uint, quantity uint, destinationWarehouseID uint) error
	Close() error
}

type GORMWarehouseRepository struct {
	DB *gorm.DB
}

func NewGORMWarehouseRepository(DBName string) (*GORMWarehouseRepository, error) {
	database, err1 := gorm.Open(sqlite.Open(DBName), &gorm.Config{})
	if err1 != nil {
		return nil, err1
	}
	err2 := database.AutoMigrate(&Warehouse{}, &Item{}, &WarehouseItems{})
	if err2 != nil {
		return nil, err2
	}
	return &GORMWarehouseRepository{DB: database}, nil
}
func (r *GORMWarehouseRepository) Close() error {
	db, err := r.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (r *GORMWarehouseRepository) CreateItem(name string, category string, description string) error {
	return r.DB.Create(&Item{Name: name, Category: category, Description: description}).Error
}

func (r *GORMWarehouseRepository) CreateWarehouse(name string, position string, capacity uint) error {
	return r.DB.Create(&Warehouse{Name: name, Position: position, Capacity: capacity}).Error
}

func (r *GORMWarehouseRepository) UpdateItem(itemID uint, name string, category string, description string) error {
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

func (r *GORMWarehouseRepository) UpdateWarehouse(warehouseID uint, name string, position string, capacity uint) error {
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

func (r *GORMWarehouseRepository) FindItemByID(itemID uint) (Item, error) {
	var item Item
	err := r.DB.First(&item, itemID).Error
	return item, err
}

func (r *GORMWarehouseRepository) FindWarehouseByID(warehouseID uint) (Warehouse, error) {
	var warehouse Warehouse
	err := r.DB.First(&warehouse, warehouseID).Error
	return warehouse, err
}

func (r *GORMWarehouseRepository) FindItemsByKeyword(keyword string) ([]Item, error) {
	var items []Item
	err := r.DB.Where("name LIKE ?", "%"+keyword+"%").Find(&items).Error
	return items, err
}

func (r *GORMWarehouseRepository) FindItemByName(name string) (Item, error) {
	var item Item
	err := r.DB.First(&item, "name = ?", name).Error
	return item, err
}

func (r *GORMWarehouseRepository) FindWarehouseByName(name string) (Warehouse, error) {
	var warehouse Warehouse
	err := r.DB.First(&warehouse, "name = ?", name).Error
	return warehouse, err
}

func (r *GORMWarehouseRepository) FindWarehousesByPosition(position string) ([]Warehouse, error) {
	var warehouses []Warehouse
	err := r.DB.Where("position = ?", position).Find(&warehouses).Error
	return warehouses, err
}

func (r *GORMWarehouseRepository) FindItemsByCategory(category string) ([]Item, error) {
	var items []Item
	err := r.DB.Where("category = ?", category).Find(&items).Error
	return items, err
}

func (r *GORMWarehouseRepository) FindItemsInWarehouse(warehouseID uint) ([]LoadedItemPack, error) {
	var correspondence []WarehouseItems
	var res []LoadedItemPack
	err1 := r.DB.Model(&WarehouseItems{}).Where("warehouse_id = ?", warehouseID).Find(&correspondence).Error
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

func (r *GORMWarehouseRepository) FindWarehousesForItem(itemID uint) ([]LoadedItemPack, error) {
	var correspondence []WarehouseItems
	var res []LoadedItemPack
	err1 := r.DB.Model(&WarehouseItems{}).Where("item_id = ?", itemID).Find(&correspondence).Error
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
func (r *GORMWarehouseRepository) DeleteItem(itemID uint) error {
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

func (r *GORMWarehouseRepository) DeleteWarehouse(warehouseID uint) error {
	var warehouse Warehouse
	var correspondence []WarehouseItems
	err1 := r.DB.First(&warehouse, warehouseID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.Model(&WarehouseItems{}).Where("warehouse_id = ?", warehouseID).Find(&correspondence).Error
	if err2 != nil {
		return err2
	}
	if len(correspondence) != 0 {
		return errors.New("warehouse is not empty")
	} else {
		return r.DB.Delete(&warehouse).Error
	}
}

func (r *GORMWarehouseRepository) SupplyItems(itemID uint, warehouseID uint, quantity uint) error {
	var item Item
	var warehouse Warehouse
	var warehouseItems []WarehouseItems
	err1 := r.DB.First(&item, itemID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.First(&warehouse, warehouseID).Error
	if err2 != nil {
		return err2
	}
	item.Quantity += quantity
	err3 := r.DB.Save(&item).Error
	if err3 != nil {
		return err3
	}
	err4 := r.DB.Model(&WarehouseItems{}).Where("item_id = ? AND warehouse_id = ?", itemID, warehouseID).Find(&warehouseItems).Error
	if err4 != nil {
		return err4
	}
	if len(warehouseItems) == 0 {
		err5 := r.DB.Create(&WarehouseItems{ItemID: itemID, WarehouseID: warehouseID, Quantity: quantity}).Error
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
	return nil
}

func (r *GORMWarehouseRepository) ConsumeItems(itemID uint, warehouseID uint, quantity uint) error {
	var item Item
	var warehouse Warehouse
	var warehouseItems []WarehouseItems
	err1 := r.DB.First(&item, itemID).Error
	if err1 != nil {
		return err1
	}
	err2 := r.DB.First(&warehouse, warehouseID).Error
	if err2 != nil {
		return err2
	}
	if item.Quantity < quantity {
		return errors.New("not enough items")
	}
	err4 := r.DB.Model(&WarehouseItems{}).Where("item_id = ? AND warehouse_id = ?", itemID, warehouseID).Find(&warehouseItems).Error
	if err4 != nil {
		return err4
	}
	if len(warehouseItems) == 0 {
		return errors.New("item not found in specified warehouse")
	} else if warehouseItems[0].Quantity < quantity {
		return errors.New("not enough items in specified warehouse")
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

func (r *GORMWarehouseRepository) TransferItems(itemID uint, sourceWarehouseID uint, quantity uint, destinationWarehouseID uint) error {
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
