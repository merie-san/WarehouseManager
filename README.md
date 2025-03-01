# Warehouse Manager
Golang project using GORM to manage a SQLite database, Gollira/mux to forward the HTTP requests to the correct handlers and various other Golang packages to implement a simple yet functional web application for managing a list warehouses containing different items. Developed for the course of Distributed Programming for Web IOT and Mobile Systems 2024/2025.

## To run the application:
Position yourself on the root directory of the project, create a data directory if it doesn't exists so the "data" files could be stored there. You can now run the application and play with it.

## To run the tests:
Create a directory "data" in the package you want to test and type "go test -v {path/to/package}". The test should run. 
