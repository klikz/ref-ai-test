package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/klikz/api_v3/internal/models"
)

func (r *Repo) LabelTemplatesGetAll() ([]models.LabelTemplate, error) {
	rows, err := r.store.db.Query(`
		SELECT t.id, t.name, t.line_id, COALESCE(ll.name, ''), t.width_mm, t.height_mm,
		       t.dpi, COALESCE(t.print_rotation_deg, 0), t.definition
		FROM lines.label_templates t
		LEFT JOIN lines.lines_list ll ON ll.line_id = t.line_id
		ORDER BY t.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.LabelTemplate{}
	for rows.Next() {
		item := models.LabelTemplate{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.LineID, &item.LineName,
			&item.WidthMm, &item.HeightMm, &item.DPI, &item.PrintRotationDeg, &item.Definition,
		); err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	return allData, rows.Err()
}

func (r *Repo) LabelTemplateGetByID(id int) (models.LabelTemplate, error) {
	item := models.LabelTemplate{}
	err := r.store.db.QueryRow(`
		SELECT t.id, t.name, t.line_id, COALESCE(ll.name, ''), t.width_mm, t.height_mm,
		       t.dpi, COALESCE(t.print_rotation_deg, 0), t.definition
		FROM lines.label_templates t
		LEFT JOIN lines.lines_list ll ON ll.line_id = t.line_id
		WHERE t.id = $1`, id).Scan(
		&item.ID, &item.Name, &item.LineID, &item.LineName,
		&item.WidthMm, &item.HeightMm, &item.DPI, &item.PrintRotationDeg, &item.Definition,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return item, errors.New("label template not found")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) LabelTemplateCreate(name string, lineID int, widthMm, heightMm float64, dpi, printRotationDeg int, definition json.RawMessage, userID int) (int, error) {
	if len(definition) == 0 {
		definition = json.RawMessage(`{"version":1,"elements":[]}`)
	}

	var id int
	err := r.store.db.QueryRow(`
		INSERT INTO lines.label_templates
			(name, line_id, width_mm, height_mm, dpi, print_rotation_deg, definition, c_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		name, lineID, widthMm, heightMm, dpi, printRotationDeg, definition, userID,
	).Scan(&id)
	return id, err
}

func (r *Repo) LabelTemplateDuplicate(sourceID int, newName string, userID int) (int, error) {
	source, err := r.LabelTemplateGetByID(sourceID)
	if err != nil {
		return 0, err
	}
	if newName == "" {
		newName = source.Name + " (nusxa)"
	}

	definition := make(json.RawMessage, len(source.Definition))
	copy(definition, source.Definition)

	return r.LabelTemplateCreate(
		newName,
		source.LineID,
		source.WidthMm,
		source.HeightMm,
		source.DPI,
		source.PrintRotationDeg,
		definition,
		userID,
	)
}

func (r *Repo) LabelTemplateUpdate(id int, name string, lineID int, widthMm, heightMm float64, dpi, printRotationDeg int, definition json.RawMessage) error {
	if len(definition) == 0 {
		existing, err := r.LabelTemplateGetByID(id)
		if err != nil {
			return err
		}
		definition = existing.Definition
	}
	if len(definition) == 0 {
		definition = json.RawMessage(`{"version":1,"elements":[]}`)
	}

	res, err := r.store.db.Exec(`
		UPDATE lines.label_templates
		SET name = $1, line_id = $2, width_mm = $3, height_mm = $4, dpi = $5,
		    print_rotation_deg = $6, definition = $7, u_time = NOW()
		WHERE id = $8`,
		name, lineID, widthMm, heightMm, dpi, printRotationDeg, definition, id,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("label template not found")
	}

	return nil
}

func (r *Repo) LabelTemplateDelete(id int) error {
	res, err := r.store.db.Exec(`DELETE FROM lines.label_templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("label template not found")
	}
	return nil
}
