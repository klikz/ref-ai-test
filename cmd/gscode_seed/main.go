package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
	_ "github.com/lib/pq"
)

const gsSeparator = "\x1d"

type modelRow struct {
	ID       int
	GS1EAN13 string
	Modeli   string
	Existing int
}

func main() {
	perModel := flag.Int("per-model", 1000, "har bir model uchun qo'shiladigan GS Code soni")
	dryRun := flag.Bool("dry-run", false, "faqat hisobot, DB ga yozmaydi")
	userID := flag.Int("user-id", 0, "c_user_id uchun foydalanuvchi ID (0 = birinchi faol user)")
	flag.Parse()

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

	models, err := loadModels(db)
	if err != nil {
		log.Fatal(err)
	}
	if len(models) == 0 {
		log.Println("modellar topilmadi")
		return
	}

	repo := store.New(db).Repo()
	resolvedUserID, err := resolveUserID(db, *userID)
	if err != nil {
		log.Fatal(err)
	}
	totalInserted := 0
	skipped := 0

	for _, model := range models {
		ean := strings.TrimSpace(model.GS1EAN13)
		if ean == "" {
			log.Printf("skip model #%d (%s): gs1_ean13 bo'sh", model.ID, model.Modeli)
			skipped++
			continue
		}
		if len(ean) != 13 {
			log.Printf("skip model #%d (%s): gs1_ean13 noto'g'ri (%q)", model.ID, model.Modeli, ean)
			skipped++
			continue
		}

		codes := generateCodes(ean, model.ID, *perModel)
		if *dryRun {
			log.Printf("dry-run model #%d (%s): %d ta kod yaratiladi (mavjud: %d)", model.ID, model.Modeli, len(codes), model.Existing)
			continue
		}

		inserted, err := repo.GsCodesBulkAdd(codes, model.ID, resolvedUserID)
		if err != nil {
			log.Fatalf("model #%d insert: %v", model.ID, err)
		}
		totalInserted += inserted
		log.Printf("model #%d (%s): +%d kod (mavjud edi: %d)", model.ID, model.Modeli, inserted, model.Existing)
	}

	if *dryRun {
		log.Printf("dry-run: %d ta model, %d ta o'tkazib yuborildi", len(models), skipped)
		return
	}

	log.Printf("tayyor: jami %d ta kod qo'shildi (%d model, %d skip)", totalInserted, len(models), skipped)
}

func loadModels(db *sql.DB) ([]modelRow, error) {
	rows, err := db.Query(`
		SELECT m.id,
			COALESCE(m.gs1_ean13, ''),
			COALESCE(m.modeli, ''),
			COALESCE(COUNT(g.id) FILTER (WHERE g.status = true), 0)::int
		FROM production.models m
		LEFT JOIN production.gscodes g ON g.model_id = m.id
		WHERE m.deleted = false
		GROUP BY m.id, m.gs1_ean13, m.modeli
		ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []modelRow{}
	for rows.Next() {
		var row modelRow
		if err := rows.Scan(&row.ID, &row.GS1EAN13, &row.Modeli, &row.Existing); err != nil {
			return items, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func resolveUserID(db *sql.DB, requested int) (int, error) {
	if requested > 0 {
		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)`, requested).Scan(&exists)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, fmt.Errorf("user #%d topilmadi", requested)
		}
		return requested, nil
	}
	var userID int
	err := db.QueryRow(`SELECT id FROM auth.users WHERE status = true ORDER BY id LIMIT 1`).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("faol foydalanuvchi topilmadi: %w", err)
	}
	return userID, nil
}

func generateCodes(ean13 string, modelID, count int) []string {
	codes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for seq := 1; len(codes) < count; seq++ {
		code := buildSeedGsCode(ean13, modelID, seq)
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes
}

func buildSeedGsCode(ean13 string, modelID, seq int) string {
	prefix := "010" + ean13
	serial := fmt.Sprintf("21M%06dS%06d", modelID, seq)
	hash := sha256.Sum256([]byte(prefix + serial))
	tail := base64.RawURLEncoding.EncodeToString(hash[:])
	if len(tail) > 40 {
		tail = tail[:40]
	}
	ai91 := fmt.Sprintf("91%04d", seq%10000)
	ai92 := "92" + tail
	return prefix + serial + gsSeparator + ai91 + gsSeparator + ai92 + "="
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)
}
