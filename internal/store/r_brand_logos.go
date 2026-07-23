package store

import (
	"strings"
)

func (r *Repo) BrandList() ([]string, error) {
	rows, err := r.store.db.Query(`
		SELECT DISTINCT TRIM(m.brend) AS brand
		FROM production.models m
		WHERE m.deleted = FALSE
		  AND TRIM(COALESCE(m.brend, '')) <> ''
		ORDER BY brand`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var brand string
		if err := rows.Scan(&brand); err != nil {
			return out, err
		}
		brand = strings.TrimSpace(brand)
		if brand != "" {
			out = append(out, brand)
		}
	}
	return out, rows.Err()
}

func (r *Repo) BrandLogosAll() (map[string]string, error) {
	rows, err := r.store.db.Query(`
		SELECT brand, logo_src
		FROM lines.brand_logos
		WHERE is_active = TRUE
		ORDER BY brand`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var brand, logoSrc string
		if err := rows.Scan(&brand, &logoSrc); err != nil {
			return out, err
		}
		brand = strings.TrimSpace(brand)
		logoSrc = strings.TrimSpace(logoSrc)
		if brand != "" && logoSrc != "" {
			out[brand] = logoSrc
		}
	}
	return out, rows.Err()
}

func (r *Repo) BrandLogoUpsert(brand, logoSrc string, userID int) error {
	brand = strings.TrimSpace(brand)
	logoSrc = strings.TrimSpace(logoSrc)
	_, err := r.store.db.Exec(`
		INSERT INTO lines.brand_logos (brand, logo_src, is_active, c_user_id)
		VALUES ($1, $2, TRUE, $3)
		ON CONFLICT (brand) DO UPDATE
		SET logo_src = EXCLUDED.logo_src,
		    is_active = TRUE,
		    u_time = NOW(),
		    u_user_id = $3`,
		brand, logoSrc, userID,
	)
	return err
}

func (r *Repo) BrandLogoDelete(brand string, userID int) error {
	brand = strings.TrimSpace(brand)
	_, err := r.store.db.Exec(`
		UPDATE lines.brand_logos
		SET is_active = FALSE,
		    u_time = NOW(),
		    u_user_id = $2
		WHERE brand = $1`,
		brand, userID,
	)
	return err
}

func ResolveBrandLogo(brand string, logos map[string]string) string {
	brand = strings.TrimSpace(brand)
	if brand == "" {
		return ""
	}
	if v, ok := logos[brand]; ok {
		return strings.TrimSpace(v)
	}
	return ""
}
