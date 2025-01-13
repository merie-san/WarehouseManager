package model

import (
	"os"
	"strconv"
	"testing"
)

var db *GORMSQLiteWarehouseRepository

func TearDown() {
	dbc, _ := db.DB.DB()
	_ = dbc.Close()
	_ = os.Remove("test.db")
}

func TestGORMRepository(t *testing.T) {
	t.Cleanup(TearDown)
	rep, err1 := NewGORMSQLiteWarehouseRepository("test.db")
	if err1 != nil {
		t.Fatalf("Reported error: %v", err1)
	}
	db = rep
	warehouses := []Warehouse{
		{Name: "Small warehouse", Position: "Florence", Capacity: 500},
		{Name: "Big warehouse", Position: "Florence", Capacity: 1200},
		{Name: "Old warehouse", Position: "Pisa", Capacity: 200},
	}
	items := []Item{
		{Name: "potatoes", Category: "vegetables", Description: "fresh potatoes from local farms"},
		{Name: "tomatoes", Category: "vegetables", Description: "imported tomatoes"},
		{Name: "toothbrush", Category: "hygiene", Description: "imported electric toothbrushes"},
		{Name: "plastic cup", Category: "table settings", Description: "high quality cup made from tritan"},
	}
	for i, v := range warehouses {
		t.Run("CreateWarehouse "+v.Name, func(t *testing.T) {
			err2 := rep.CreateWarehouse(v.Name, v.Position, v.Capacity)
			if err2 != nil {
				t.Fatalf("Reported error: %v", err2)
			}
			var temp Warehouse
			err3 := rep.DB.First(&temp, i+1).Error
			if err3 != nil {
				t.Errorf("Warehouse %s not correctly created\nerror message: %v", v.Name, err3)
			}
			if temp.Name != v.Name {
				t.Errorf("Warehouse name is not correct.\nexpected name: %s actual name: %s", v.Name, temp.Name)
			}
			if temp.Position != v.Position {
				t.Errorf("Warehouse position is not correct.\nexpected position: %s actual position: %s", v.Position, temp.Position)
			}
			if temp.Capacity != v.Capacity {
				t.Errorf("Warehouse capacity is not correct.\nexpected capacity: %d actual capacity: %d", v.Capacity, temp.Capacity)
			}
		})
	}
	for i, v := range items {
		t.Run("CreateItem "+v.Name, func(t *testing.T) {
			err2 := rep.CreateItem(v.Name, v.Category, v.Description)
			if err2 != nil {
				t.Fatalf("Reported error: %v", err2)
			}
			var temp Item
			err3 := rep.DB.First(&temp, i+1).Error
			if err3 != nil {
				t.Errorf("Item %s not correctly created\nerror message: %v", v.Name, err3)
			}
			if temp.Name != v.Name {
				t.Errorf("Item name is not correct.\nexpected name: %s actual name: %s", v.Name, temp.Name)
			}
			if temp.Category != v.Category {
				t.Errorf("Item category is not correct.\nexpected category: %s actual category: %s", v.Category, temp.Category)
			}
			if temp.Description != v.Description {
				t.Errorf("Item description is not correct.\nexpected description: %s actual description: %s", v.Description, temp.Description)
			}
			if temp.Quantity != 0 {
				t.Errorf("Item quantity is not initialized at zero")
			}
		})
	}
	for i := 1; i <= 2; i++ {
		t.Run("FindItemByID "+strconv.Itoa(i), func(t *testing.T) {
			temp, err2 := rep.FindItemByID(uint(i))
			if err2 != nil {
				t.Fatalf("Reported error: %v", err2)
			}
			if temp.Name != items[i-1].Name {
				t.Errorf("Item name is not correct.\nexpected name: %s actual name: %s", items[i-1].Name, temp.Name)
			}
			if temp.Category != items[i-1].Category {
				t.Errorf("Item category is not correct.\nexpected category: %s actual category: %s", items[i-1].Category, temp.Category)
			}
			if temp.Description != items[i-1].Description {
				t.Errorf("Item description is not correct.\nexpected description: %s actual description: %s", items[i-1].Description, temp.Description)
			}
		})
		t.Run("FindWarehouseByID "+strconv.Itoa(i), func(t *testing.T) {
			temp, err2 := rep.FindWarehouseByID(uint(i))
			if err2 != nil {
				t.Fatalf("Reported error: %v", err2)
			}
			if temp.Name != warehouses[i-1].Name {
				t.Errorf("Warehouse name is not correct.\nexpected name: %s actual name: %s", warehouses[i-1].Name, temp.Name)
			}
			if temp.Position != warehouses[i-1].Position {
				t.Errorf("Warehouse position is not correct.\nexpected position: %s actual position: %s", warehouses[i-1].Position, temp.Position)
			}
			if temp.Capacity != warehouses[i-1].Capacity {
				t.Errorf("Warehouse capacity is not correct.\nexpected capacity: %d actual capacity: %d", warehouses[i-1].Capacity, temp.Capacity)
			}
		})
	}
	for i, v := range items {
		t.Run("FindItemsByName "+v.Name, func(t *testing.T) {
			temp, err2 := rep.FindItemByName(v.Name)
			if err2 != nil {
				t.Fatalf("Reported error: %v", err2)
			}
			if len(temp) == 0 {
				t.Errorf("No item with expected name found")
			} else if int(temp[0].ID) != i+1 {
				t.Errorf("Incorrect item ID.\nexpected ID: %d actual ID: %d", i+1, temp[0].ID)
			}
		})
	}
	for i, v := range warehouses {
		t.Run("FindWarehousesByName "+v.Name, func(t *testing.T) {
			temp, err2 := rep.FindWarehouseByName(v.Name)
			if err2 != nil {
				t.Fatalf("Reported error: %v", err2)
			}
			if len(temp) == 0 {
				t.Errorf("No warehouse with expected name found")
			} else if int(temp[0].ID) != i+1 {
				t.Errorf("Incorrect warehouse ID.\nexpected ID: %d actual ID: %d", i+1, temp[0].ID)
			}
		})
	}
	t.Run("FindItemByKeyword", func(t *testing.T) {
		temp, err2 := rep.FindItemsByKeyword("imported")
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		if len(temp) != 2 {
			t.Errorf("Incorrect number of items found.\nexpected number: 2 actual number: %d", len(temp))
		}
		for _, v := range temp {
			if v.Name != "tomatoes" && v.Name != "toothbrush" {
				t.Errorf("Incorrect item found: %s expected items: tomatoes, toothbrush", v.Name)
			}
		}
		temp2, err3 := rep.FindItemsByKeyword("cheap")
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if len(temp2) != 0 {
			t.Errorf("Incorrect number of items found.\nexpected number: 0 actual number: %d", len(temp2))
		}
	})
	t.Run("FindWarehouseByPosition", func(t *testing.T) {
		temp, err2 := rep.FindWarehousesByPosition("Florence")
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		if len(temp) != 2 {
			t.Errorf("Incorrect number of warehouses found.\nexpected number: 2 actual number: %d", len(temp))
		}
		for _, v := range temp {
			if v.Name != "Small warehouse" && v.Name != "Big warehouse" {
				t.Errorf("Incorrect warehouse found: %s expected warehouses: Small warehouse, Big warehouse", v.Name)
			}
		}
		temp2, err3 := rep.FindWarehousesByPosition("Rome")
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if len(temp2) != 0 {
			t.Errorf("Incorrect number of warehouses found.\nexpected number: 0 actual number: %d", len(temp2))
		}
	})
	t.Run("FindItemByCategory", func(t *testing.T) {
		temp, err2 := rep.FindItemsByCategory("hygiene")
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		if len(temp) != 1 {
			t.Errorf("Incorrect number of items found.\nexpected number: 1 actual number: %d", len(temp))
		}
		if temp[0].Name != "toothbrush" {
			t.Errorf("Incorrect item found: %s expected item: toothbrush", temp[0].Name)
		}
		temp2, err3 := rep.FindItemsByCategory("electronics")
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if len(temp2) != 0 {
			t.Errorf("Incorrect number of items found.\nexpected number: 0 actual number: %d", len(temp2))
		}
	})
	t.Run("UpdateItem", func(t *testing.T) {
		err2 := rep.UpdateItem(1, "potatoes", "vegetables", "agata potatoes")
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp Item
		err3 := rep.DB.First(&temp, 1).Error
		if err3 != nil {
			t.Errorf("Item %s not correctly updated\nerror message: %v", "potatoes", err3)
		}
		if temp.Description != "agata potatoes" || temp.Name != "potatoes" || temp.Category != "vegetables" {
			t.Errorf("Item wasn't updated correctly\nexpected values-\ndescription: \"agata potatoes\", name: \"potatoes\", category: \"vegetables\"\nactual values-\ndescription: \"%s\", name: \"%s\", category: \"%s\"", temp.Description, temp.Name, temp.Category)
		}
	})
	t.Run("UpdateWarehouse", func(t *testing.T) {
		err2 := rep.UpdateWarehouse(2, "Big warehouse", "Florence", 1500)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp Warehouse
		err3 := rep.DB.First(&temp, 2).Error
		if err3 != nil {
			t.Errorf("Warehouse %s not correctly updated\nerror message: %v", "Small warehouse", err3)
		}
		if temp.Name != "Big warehouse" || temp.Position != "Florence" || temp.Capacity != 1500 {
			t.Errorf("Warehouse wasn't updated correctly\nexpected values-\nname: \"Big warehouse\", position: \"Florence\", capacity: 1500\nactual values-\nname: \"%s\", position: \"%s\", capacity: %d", temp.Name, temp.Position, temp.Capacity)
		}
	})
	t.Run("DeleteItem", func(t *testing.T) {
		err2 := rep.DeleteItem(4)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp []Item
		err3 := rep.DB.Find(&temp, 4).Error
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if len(temp) != 0 {
			t.Errorf("Item wasn't deleted correctly")
		}
	})
	t.Run("DeleteWarehouse", func(t *testing.T) {
		err2 := rep.DeleteWarehouse(3)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp []Warehouse
		err3 := rep.DB.Find(&temp, 3).Error
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if len(temp) != 0 {
			t.Errorf("Warehouse wasn't deleted correctly")
		}
	})
	t.Run("SupplyItems", func(t *testing.T) {
		err2 := rep.SupplyItems(1, 1, 100)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp Item
		err3 := rep.DB.First(&temp, 1).Error
		if err3 != nil {
			t.Fatalf("Item %s not correctly updated\nerror message: %v", "potatoes", err3)
		}
		if temp.Quantity != 100 {
			t.Errorf("Items weren't supplied correctly\nexpected quantity: 100\nactual quantity: %d", temp.Quantity)
		}
		var temp2 WarehouseItem
		err4 := rep.DB.First(&temp2, "item_id = ? AND warehouse_id = ?", 1, 1).Error
		if err4 != nil {
			t.Fatalf("WarehouseItem small_warehouse-potatoes wasn't correctly updated, error message: %v", err4)
		}
		if temp2.Quantity != 100 {
			t.Errorf("WarehouseItem small_warehouse-potatoes quantity wasn't updated correctly\nexpected quantity: 100\nactual quantity: %d", temp2.Quantity)
		}
		err5 := rep.SupplyItems(2, 1, 100)
		if err5 != nil {
			t.Fatalf("Reported error: %v", err5)
		}
		var temp3 Item
		err6 := rep.DB.First(&temp3, 2).Error
		if err6 != nil {
			t.Fatalf("Item %s not correctly updated\nerror message: %v", "tomatoes", err6)
		}
		if temp3.Quantity != 100 {
			t.Errorf("Items weren't supplied correctly\nexpected quantity: 100\nactual quantity: %d", temp3.Quantity)
		}
		var temp4 WarehouseItem
		err7 := rep.DB.First(&temp4, "item_id = ? AND warehouse_id = ?", 2, 1).Error
		if err7 != nil {
			t.Fatalf("WarehouseItem small_warehouse-tomatoes wasn't correctly updated, error message: %v", err7)
		}
		if temp4.Quantity != 100 {
			t.Errorf("WarehouseItem small_warehouse-tomatoes quantity wasn't updated correctly\nexpected quantity: 100\nactual quantity: %d", temp4.Quantity)
		}
		err8 := rep.SupplyItems(2, 2, 20)
		if err8 != nil {
			t.Fatalf("Reported error: %v", err8)
		}
		var temp5 Item
		err9 := rep.DB.First(&temp5, 2).Error
		if err9 != nil {
			t.Fatalf("Item %s not correctly updated\nerror message: %v", "tomatoes", err9)
		}
		if temp5.Quantity != 120 {
			t.Errorf("Items weren't supplied correctly\nexpected quantity: 120\nactual quantity: %d", temp5.Quantity)
		}
		var temp6 WarehouseItem
		err10 := rep.DB.First(&temp6, "item_id = ? AND warehouse_id = ?", 2, 2).Error
		if err10 != nil {
			t.Fatalf("WarehouseItem big_warehouse-tomatoes wasn't correctly updated, error message: %v", err10)
		}
		if temp6.Quantity != 20 {
			t.Errorf("WarehouseItem big_warehouse-tomatoes quantity wasn't updated correctly\nexpected quantity: 20\nactual quantity: %d", temp6.Quantity)
		}
	})
	t.Run("DeleteNotEmpty", func(t *testing.T) {
		err2 := rep.DeleteItem(1)
		if err2 == nil {
			t.Errorf("No error reported when deleting item with non-empty item")
		} else {
			if err2.Error() != "item is not empty" {
				t.Errorf("unexpected error message")
			}
		}
		err3 := rep.DeleteWarehouse(1)
		if err3 == nil {
			t.Errorf("No error reported when deleting warehouse with non-empty warehouse")
		} else {
			if err3.Error() != "warehouse is not empty" {
				t.Errorf("unexpected error message: %s", err3.Error())
			}
		}
	})
	t.Run("FindItemsInWarehouse", func(t *testing.T) {
		temp, err2 := rep.FindItemsInWarehouse(1)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		if len(temp) != 2 {
			t.Errorf("Incorrect number of LoadedItemPacks found.\nexpected number: 2 actual number: %d", len(temp))
		}
		if temp[0].ItemID != 1 || temp[0].WarehouseID != 1 || temp[0].ItemName != "potatoes" || temp[0].ItemQuantity != 100 {
			t.Errorf("Incorrect LoadedItemPack found\nexpected LoadedItemPacks: potatoes, 100, Small warehouse\nactual LoadedItemPacks: %s, %v, %s", temp[0].ItemName, temp[0].ItemQuantity, temp[0].WarehouseName)
		}
		if temp[1].ItemID != 2 || temp[1].WarehouseID != 1 || temp[1].ItemName != "tomatoes" || temp[1].ItemQuantity != 100 {
			t.Errorf("Incorrect LoadedItemPack found\nexpected LoadedItemPacks: tomatoes, 100, Small warehouse\nactual LoadedItemPacks: %s, %v, %s", temp[1].ItemName, temp[1].ItemQuantity, temp[1].WarehouseName)
		}
	})
	t.Run("FindWarehousesForItem", func(t *testing.T) {
		temp, err2 := rep.FindWarehousesForItem(2)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		if len(temp) != 2 {
			t.Errorf("Incorrect number of warehouses found.\nexpected number: 2 actual number: %d", len(temp))
		}
		if temp[0].ItemID != 2 || temp[0].WarehouseID != 1 || temp[0].WarehouseName != "Small warehouse" || temp[0].WarehousePosition != "Florence" || temp[0].ItemQuantity != 100 {
			t.Errorf("Incorrect LoadedItemPack found\nexpected LoadedItemPack: tomatoes, Small warehouse, 100\nactual LoadedItemPack: %s, %s, %v", temp[0].ItemName, temp[0].WarehouseName, temp[0].ItemQuantity)
		}
		if temp[1].ItemID != 2 || temp[1].WarehouseID != 2 || temp[1].WarehouseName != "Big warehouse" || temp[1].WarehousePosition != "Florence" || temp[1].ItemQuantity != 20 {
			t.Errorf("Incorrect LoadedItemPack found\nexpected LoadedItemPack: tomatoes, Big warehouse, 20\nactual LoadedItemPack: %s, %s, %v", temp[1].ItemName, temp[1].WarehouseName, temp[1].ItemQuantity)
		}
	})
	t.Run("SupplyItemsFull", func(t *testing.T) {
		err2 := rep.SupplyItems(1, 1, 1000)
		if err2 == nil {
			t.Errorf("No error reported when supplying more items than warehouse's capacity")
		} else {
			if err2.Error() != "warehouse is full: 1200 > 500" {
				t.Errorf("unexpected error message: %s", err2.Error())
			}
		}
	})
	t.Run("ConsumeItems", func(t *testing.T) {
		err2 := rep.ConsumeItems(1, 1, 10)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp Item
		err3 := rep.DB.First(&temp, 1).Error
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if temp.Quantity != 90 {
			t.Errorf("Items weren't consumed correctly\nexpected quantity: 90\nactual quantity: %d", temp.Quantity)
		}
		var temp2 WarehouseItem
		err5 := rep.DB.First(&temp2, "item_id = ? AND warehouse_id = ?", 1, 1).Error
		if err5 != nil {
			t.Fatalf("Reported message: %v", err5)
		}
		if temp2.Quantity != 90 {
			t.Errorf("WarehouseItem wasn't updated correctly\nexpected quantity: 90\nactual quantity: %d", temp2.Quantity)
		}
		err4 := rep.ConsumeItems(1, 1, 100)
		if err4 == nil {
			t.Errorf("No error reported when consuming more items than warehouse's quantity")
		} else {
			if err4.Error() != "not enough items: 90 < 100" {
				t.Errorf("unexpected error message: %s", err4.Error())
			}
		}
	})
	t.Run("TransferItems", func(t *testing.T) {
		err2 := rep.TransferItems(1, 1, 10, 2)
		if err2 != nil {
			t.Fatalf("Reported error: %v", err2)
		}
		var temp Item
		err3 := rep.DB.First(&temp, 1).Error
		if err3 != nil {
			t.Fatalf("Reported error: %v", err3)
		}
		if temp.Quantity != 90 {
			t.Errorf("Items weren't transferred correctly\nexpected quantity: 90\nactual quantity: %d", temp.Quantity)
		}
		var temp2 WarehouseItem
		err5 := rep.DB.First(&temp2, "item_id = ? AND warehouse_id = ?", 1, 1).Error
		if err5 != nil {
			t.Fatalf("Reported message: %v", err5)
		}
		if temp2.Quantity != 80 {
			t.Errorf("WarehouseItem wasn't updated correctly\nexpected quantity: 80\nactual quantity: %d", temp2.Quantity)
		}
		var temp3 WarehouseItem
		err6 := rep.DB.First(&temp3, "item_id = ? AND warehouse_id = ?", 1, 2).Error
		if err6 != nil {
			t.Fatalf("Reported message: %v", err6)
		}
		if temp3.Quantity != 10 {
			t.Errorf("WarehouseItem wasn't updated correctly\nexpected quantity: 10\nactual quantity: %d", temp3.Quantity)
		}
	})
}
