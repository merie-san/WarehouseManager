package main

import (
	"WarehouseManager/internal/model"
	"fmt"
)

func main() {
	rep, err1 := model.NewGORMSQLiteWarehouseRepository("test.db")
	if err1 != nil {
		panic(err1)
	}
	err2 := rep.CreateWarehouse("testWarehouse", "testPosition", 100)
	if err2 != nil {
		panic(err2)
	}
	err5 := rep.CreateWarehouse("testWarehouse2", "testPosition2", 100)
	if err5 != nil {
		panic(err5)
	}
	err3 := rep.CreateItem("testItem", "testCategory", "testDescription")
	if err3 != nil {
		panic(err3)
	}
	err4 := rep.SupplyItems(1, 1, 90)
	if err4 != nil {
		panic(err4)
	}
	err9 := rep.SupplyItems(1, 1, 10)
	if err9 != nil {
		panic(err9)
	}
	err6 := rep.TransferItems(1, 1, 110, 2)
	if err6 != nil {
		panic(err6)
	}
	items, err8 := rep.FindItemsInWarehouse(1)
	if err8 != nil {
		panic(err8)
	}
	for _, v := range items {
		fmt.Printf("%#v\n", v)
	}
	items2, err7 := rep.FindItemsInWarehouse(2)
	if err7 != nil {
		panic(err7)
	}
	for _, v := range items2 {
		fmt.Printf("%#v\n", v)
	}
	defer rep.Close()
}
