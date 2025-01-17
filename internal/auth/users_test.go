package auth

import (
	"WarehouseManager/internal/model"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"testing"
)

type testUser struct {
	userID   uint
	username string
	password string
	database string
}

func TestUserSessionManager(t *testing.T) {
	var manager *AuthenticationManager
	t.Run("LoadAuthManager", func(t *testing.T) {
		var err1 error
		manager, err1 = LoadAuthManager(func(name string) (model.WarehouseRepository, error) {
			return model.NewGORMSQLiteWarehouseRepository(name)
		})
		if err1 != nil {
			t.Fatalf("Reported error: %v", err1)
		}
		_, err2 := os.Stat("users.json")
		if err2 != nil {
			t.Fatalf("users.json file not found")
		}
		if manager.Users == nil {
			t.Fatalf("Users not initialized")
		}
	})
	t.Cleanup(func() {
		err := manager.DeleteAllUsers()
		if err != nil {
			t.Fatalf("Reported error: %v", err)
		}
		dirEntries, err1 := os.ReadDir(".")
		if err1 != nil {
			t.Fatalf("Failed to read directory\nerror: %v", err1)
		}
		for _, v := range dirEntries {
			matching, err2 := regexp.MatchString(`usr[0-9]+\.db`, v.Name())
			if err2 != nil {
				t.Fatalf("Failed to match file name\nerror: %v", err2)
			}
			if matching {
				err3 := os.Remove(v.Name())
				if err3 != nil {
					t.Fatalf("Failed to remove file\nerror: %v", err3)
				}
			}
		}
	})
	testUsers := []testUser{
		{userID: 0, username: "user1", password: "<PASSWORD>", database: "usr0.db"},
		{userID: 1, username: "user2", password: "<PASSWORD>", database: "usr1.db"},
		{userID: 2, username: "user3", password: "pass1hello", database: "usr2.db"},
	}
	for i, v := range testUsers {
		t.Run("Register user"+strconv.Itoa(i), func(t *testing.T) {
			err1 := manager.Register(v.username, v.password)
			if err1 != nil {
				t.Fatalf("Failed Registration of user: %s\nerror: %v", v.username, err1)
			}
			if manager.Users[i].UserID != v.userID || manager.Users[i].Username != v.username || manager.Users[i].EncryptedPassword != ShaHashing(v.password) || manager.Users[i].AssignedDatabase != v.database {
				t.Errorf("User not registered correctly\nexpected values- username: %s, encrypted password: %s, database %s, user ID %d\nactual values- username: %s, encrypted password: %s, database %s, user ID %d",
					v.username, ShaHashing(v.password), v.database, v.userID, manager.Users[i].Username, manager.Users[i].EncryptedPassword, manager.Users[i].AssignedDatabase, manager.Users[i].UserID)
			}
			file, err2 := os.Open("users.json")
			if err2 != nil {
				t.Fatalf("Failed to open users.json file\nerror: %v", err2)
			}
			defer file.Close()
			decoder := json.NewDecoder(file)
			users := make([]User, 0)
			err3 := decoder.Decode(&users)
			if err3 != nil {
				t.Fatalf("Failed to decode users.json file\nerror: %v", err3)
			}
			if users[i].UserID != v.userID || users[i].Username != v.username || users[i].EncryptedPassword != ShaHashing(v.password) || users[i].AssignedDatabase != v.database {
				t.Errorf("User not registered correctly\nexpected values- username: %s, encrypted password: %s, database %s, user ID %d\nactual values- username: %s, encrypted password: %s, database %s, user ID %d",
					v.username, ShaHashing(v.password), v.database, v.userID, users[i].Username, users[i].EncryptedPassword, users[i].AssignedDatabase, users[i].UserID)
			}
		})
	}
	t.Run("Register edge cases", func(t *testing.T) {
		err1 := manager.Register("user1", "pass1")
		if err1 == nil {
			t.Errorf("No error reported when registering user with existing username")
		}
		if err1.Error() != "username already exists" {
			t.Errorf("Unexpected error message")
		}
		err2 := manager.Register("user4", "pass4")
		if err2 == nil {
			t.Errorf("No error reported when registering user with invalid password")
		}
		if err2.Error() != "password must be at least 8 characters long" {
			t.Errorf("unexpected error message: %v", err2)
		}
	})
	t.Run("Operations without login", func(t *testing.T) {
		err := manager.Logout("user1")
		if err == nil {
			t.Errorf("No error reported when logging out without having first logged in")
		}
		err2 := manager.DeleteItem(0, 1)
		if err2 == nil {
			t.Errorf("No error reported when deleting item without having first logged in")
		}
		_, err3 := manager.FindItemsByKeyword(0, "Hello")
		if err3 == nil {
			t.Errorf("No error reported when searching for items without having first logged in")
		}
	})
	t.Run("Login", func(t *testing.T) {
		if manager.IsLoggedIn("user1") {
			t.Errorf("User sholdn't be logged in yet")
		}
		id, err1 := manager.Login("user1", "<PASSWORD>")
		if err1 != nil {
			t.Fatalf("Failed to login user: %s\nerror: %v", "user1", err1)
		}
		if id != 0 {
			t.Errorf("Returned user ID is wrong")
		}
		if len(manager.ActiveUsers) == 0 {
			t.Fatalf("CurrentUser not initialized")
		}
		if manager.ActiveUsers[0].User.Username != "user1" {
			t.Errorf("CurrentUser not initialized correctly\nexpected username: user1, actual username: %s", manager.ActiveUsers[0].User.Username)
		}
		if manager.ActiveUsers[0].User.UserID != 0 {
			t.Errorf("CurrentUser not initialized correctly\nexpected user ID: 0, actual user ID: %d", manager.ActiveUsers[0].User.UserID)
		}
		if manager.ActiveUsers[0].User.AssignedDatabase != "usr0.db" {
			t.Errorf("CurrentUser not initialized correctly\nexpected database: usr0.db, actual database: %s", manager.ActiveUsers[0].User.AssignedDatabase)
		}
		if !manager.IsLoggedIn("user1") {
			t.Errorf("User should be logged in")
		}
		_, err2 := manager.Login("user1", "<PASSWORD>")
		if err2 == nil {
			t.Errorf("No error reported when logging in two times")
		}
	})
	t.Run("Logout", func(t *testing.T) {
		err := manager.Logout("user1")
		if err != nil {
			t.Fatalf("Failed to logout user\nerror: %v", err)
		}
		if len(manager.ActiveUsers) != 0 {
			t.Errorf("CurrentUser not cleared correctly")
		}
	})
	t.Run("ChangePassword", func(t *testing.T) {
		_, err := manager.Login("user1", "<PASSWORD>")
		if err != nil {
			t.Fatalf("Failed to login user, error: %v", err)
		}
		err1 := manager.ChangePassword("user1", "<PASSWORD>", "Hello world!")
		if err1 != nil {
			t.Fatalf("Failed to change password, error: %v", err1)
		}
		err3 := manager.Logout("user1")
		if err3 != nil {
			t.Fatalf("Failed to logout user, error: %v", err3)
		}
		_, err4 := manager.Login("user1", "Hello world!")
		if err4 != nil {
			t.Errorf("Password hasn't been set to the specified value")
		}
	})
	t.Run("Independence of user repositories", func(t *testing.T) {
		err1 := manager.CreateItem(0, "Sunglasses", "accessories", "black stylish sunglasses")
		if err1 != nil {
			t.Fatalf("Failed to create item, error: %v", err1)
		}
		err2 := manager.Logout("user1")
		if err2 != nil {
			t.Fatalf("Failed to logout user, error: %v", err2)
		}
		_, err3 := manager.Login("user2", "<PASSWORD>")
		if err3 != nil {
			t.Errorf("Password hasn't been set to the specified value")
		}
		item, err4 := manager.FindItemByName(1, "Sunglasses")
		if err4 != nil {
			t.Fatalf("Unexpected error: %v", err4)
		}
		if len(item) != 0 {
			t.Errorf("Created item was found in user2's repository")
		}
		err5 := manager.Logout("user2")
		if err5 != nil {
			t.Fatalf("Failed to logout user, error: %v", err5)
		}
		_, err6 := manager.Login("user1", "Hello world!")
		if err6 != nil {
			t.Errorf("Password hasn't been set to the specified value")
		}
		item2, err7 := manager.FindItemsByCategory(0, "accessories")
		if err7 != nil {
			t.Fatalf("APPError when retrieving item: %v", err7)
		}
		if len(item2) != 1 {
			t.Errorf("Created item was not found in user1's repository")
		}
		if item2[0].Name != "Sunglasses" || item2[0].Category != "accessories" || item2[0].Description != "black stylish sunglasses" {
			t.Errorf("Retrieved has modified values\nexpected values - name: Sunglasses, category: accessories, description: black stylish sunglasses\nactual values - name: %s, category: %s, description: %s",
				item2[0].Name, item2[0].Category, item2[0].Description)
		}
	})
}
