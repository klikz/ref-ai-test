package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/klikz/api_v3/utils"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	connString, err := utils.DBConnString()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	printRows(db, "database", `
		SELECT current_database(), current_date::text,
			(SELECT COUNT(*) FROM auth.users WHERE status = true)::text`)
	printRows(db, "printers", `
		SELECT p.line_id::text, p.id::text, p.printer_name,
			COALESCE(p.label_template_id, 0)::text, COALESCE(lt.name, '')
		FROM lines.printers_v2 p
		LEFT JOIN lines.label_templates lt ON lt.id = p.label_template_id
		WHERE p.line_id BETWEEN 4 AND 9
		ORDER BY p.line_id, p.id`)
	printRows(db, "models", `
		SELECT m.id::text, COALESCE(m.seriya_raqami, ''), COALESCE(m.door_code, ''),
			COALESCE(m.modeli, ''), COUNT(g.id) FILTER (WHERE g.status = true)::text
		FROM production.models m
		LEFT JOIN production.gscodes g ON g.model_id = m.id
		WHERE m.deleted = false
		GROUP BY m.id
		ORDER BY COUNT(g.id) FILTER (WHERE g.status = true) DESC, m.id
		LIMIT 12`)
	printRows(db, "fin_press", `
		SELECT fpc.id::text, fpc.component_id::text,
			COALESCE(c.manufacturer_code, ''), COALESCE(c.standard_name_uz, '')
		FROM production.fin_press_components fpc
		JOIN production.components c ON c.id = fpc.component_id
		ORDER BY fpc.id
		LIMIT 10`)
	printRows(db, "radiator", `
		SELECT rc.id::text, rc.component_id::text, rc.fin_press_component_id::text,
			COALESCE(rc.seriya_raqami, ''), COALESCE(c.manufacturer_code, '')
		FROM production.radiator_components rc
		JOIN production.components c ON c.id = rc.component_id
		ORDER BY rc.id
		LIMIT 10`)
}

func printRows(db *sql.DB, label, query string) {
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("%s: %v", label, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[%s]\n", label)
	for rows.Next() {
		values := make([]string, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			log.Fatal(err)
		}
		fmt.Println(values)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
