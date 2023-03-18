package api

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strings"
)

const clearAllQuery = `TRUNCATE public."Size", public."Brand", public."Clothes", public."Size", public."Record", public."ClothesAndCategory", public."Category" CASCADE;`

func ClearAll(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("postgres", getConnectionUrl())
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	_, err = db.Exec(clearAllQuery)
	if err != nil {
		log.Println(err)
	}
}

const SelectAllQuery = `
SELECT Cl.id, Cl.name, B.name, quantity, C.name, S.name
FROM "Clothes" as Cl
INNER JOIN "Brand" B on B.id = Cl.brand_id
INNER JOIN "Record" R on Cl.id = R.clothes_id
INNER JOIN "Size" S on R.size_id = S.id
INNER JOIN "ClothesAndCategory" CAC on Cl.id = CAC.clothes_id
INNER JOIN "Category" C on CAC.category_id = C.id;
`

func GetAll(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("postgres", getConnectionUrl())
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	rows, err := db.Query(SelectAllQuery)
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	var records []string
	for rows.Next() {
		record := Record{}
		rows.Scan(&record.ClothesId, &record.ClothesName, &record.BrandName, &record.Quantity, &record.CategoryName, &record.SizeName)

		records = append(records, fmt.Sprint(record))
	}

	fmt.Fprintf(w, "%s", strings.Join(records, "\n"))
}
