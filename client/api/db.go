package api

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

func getConnectionUrl() string {
	return os.Getenv("DB_PATH")
}

func GetAll() []Record {
	db, err := sql.Open("sqlite3", getConnectionUrl())
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT clothes_id, clothes_name, brand_name, category_name, quantity, size_name  FROM shop")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		record := Record{}
		err := rows.Scan(&record.ClothesId, &record.ClothesName, &record.BrandName, &record.CategoryName, &record.Quantity, &record.SizeName)
		if err != nil {
			log.Println(err)
		}

		records = append(records, record)
	}

	return records
}
