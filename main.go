package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Replace with your database connection details
	dsn := "user:pwd@tcp(host:port)/db" // Open a database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	// Query for table names in the database
	tableNames, err := getTableNames(db)
	if err != nil {
		log.Fatalf("Error fetching table names: %v", err)
	}

	// Iterate through table names and generate models
	for _, tableName := range tableNames {
		modelCode := generateModelCode(db, tableName)
		modelFileName := fmt.Sprintf("%s.go", strings.ToLower(tableName))
		saveModelToFile(modelFileName, modelCode)
		fmt.Printf("Generated model for table: %s\n", tableName)
	}
}

func getTableNames(db *sql.DB) ([]string, error) {
	query := "SHOW TABLES"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, tableName)
	}

	return tableNames, nil
}

type NullString struct {
	sql.NullString
}

func (ns NullString) String() string {
	if ns.Valid {
		return ns.NullString.String
	}
	return "NULL"
}

func generateModelCode(db *sql.DB, tableName string) string {
	query := fmt.Sprintf("DESCRIBE %s", tableName)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Error fetching table schema for %s: %v", tableName, err)
	}
	defer rows.Close()

	var columnName, dataType, nullable, key, extra string
	// var columns []string
	var columnDefault NullString // Use custom type to handle NULL values
	var columnNames []string
	for rows.Next() {

		fmt.Println(rows)
		if err := rows.Scan(&columnName, &dataType, &nullable, &key, &columnDefault, &extra); err != nil {
			log.Fatalf("Error scanning schema for %s: %v", tableName, err)
		}
		// Define Go field and tag based on the column information
		goField := columnName
		goTag := fmt.Sprintf("orm:\"column(%s);", columnName)
		if key == "PRI" {
			goTag += "pk;"
		}
		if extra == "auto_increment" {
			goTag += "auto;"
		}
		if nullable == "YES" {
			goTag += "null;"
		}
		if strings.Contains(dataType, "varchar") {
			dataType = "string"
		} else if strings.Contains(dataType, "int") {
			dataType = "int"
		} else if strings.Contains(dataType, "enum") {
			dataType = "string"
		} else if strings.Contains(dataType, "time") {
			dataType = "time.Time"
		}
		goTag += "\""

		// Construct the Go field and tag
		columnDefinition := fmt.Sprintf("%s %s ", strings.Title(goField), dataType)
		if goTag != "orm:\"column();\"" {
			columnDefinition += "`" + goTag + " json:\"" + goField + "\"`"
		}

		columnNames = append(columnNames, columnDefinition)
	}
	title := strings.Title(tableName)
	// Generate the Go model code
	modelCode := fmt.Sprintf(`package packagemodel

import (
	"time"
    "github.com/astaxie/beego/orm"
)

type %s struct {
    %s
}

func (t *%s) TableName() string {
    return "%s"
}

func init() {
    orm.RegisterModel(new(%s))
}
`, title, strings.Join(columnNames, "\n    "), title, tableName, title)

	return modelCode
}

func saveModelToFile(fileName, modelCode string) {
	folder := "models"
	filePath := fmt.Sprintf("%s/%s", folder, fileName) // Combine folder and file name
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Error creating file %s: %v", filePath, err)
	}
	defer file.Close()

	_, err = file.WriteString(modelCode)
	if err != nil {
		log.Fatalf("Error writing to file %s: %v", filePath, err)
	}
}
