package store

import (
	"database/sql"
	"errors"

	"github.com/klikz/api_v3/internal/models"
)

func (r *Repo) PrintersV2GetAll() ([]models.PrinterV2, error) {
	rows, err := r.store.db.Query(`
		SELECT p.id, p.line_id, ll.name, p.printer_name, p.address,
		       COALESCE(p.label_template_id, 0),
		       COALESCE(lt.name, ''),
		       COALESCE(p.print_language, 'gdi')
		FROM lines.printers_v2 p
		INNER JOIN lines.lines_list ll ON ll.line_id = p.line_id
		LEFT JOIN lines.label_templates lt ON lt.id = p.label_template_id
		ORDER BY p.line_id, p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.PrinterV2{}
	for rows.Next() {
		item := models.PrinterV2{}
		if err := rows.Scan(
			&item.ID, &item.LineID, &item.LineName, &item.PrinterName, &item.Address,
			&item.LabelTemplateID, &item.LabelTemplateName, &item.PrintLanguage,
		); err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	return allData, rows.Err()
}

func (r *Repo) PrintersV2GetByLine(lineID int) ([]models.PrinterV2, error) {
	if lineID <= 0 {
		return nil, errors.New("line_id noto'g'ri")
	}

	rows, err := r.store.db.Query(`
		SELECT p.id, p.line_id, ll.name, p.printer_name, p.address,
		       COALESCE(p.label_template_id, 0),
		       COALESCE(lt.name, ''),
		       COALESCE(p.print_language, 'gdi')
		FROM lines.printers_v2 p
		INNER JOIN lines.lines_list ll ON ll.line_id = p.line_id
		LEFT JOIN lines.label_templates lt ON lt.id = p.label_template_id
		WHERE p.line_id = $1
		ORDER BY p.id`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.PrinterV2{}
	for rows.Next() {
		item := models.PrinterV2{}
		if err := rows.Scan(
			&item.ID, &item.LineID, &item.LineName, &item.PrinterName, &item.Address,
			&item.LabelTemplateID, &item.LabelTemplateName, &item.PrintLanguage,
		); err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	return allData, rows.Err()
}

func (r *Repo) PrinterV2GetByID(id int) (models.PrinterV2, error) {
	item := models.PrinterV2{}
	err := r.store.db.QueryRow(`
		SELECT p.id, p.line_id, ll.name, p.printer_name, p.address,
		       COALESCE(p.label_template_id, 0),
		       COALESCE(lt.name, ''),
		       COALESCE(p.print_language, 'gdi')
		FROM lines.printers_v2 p
		INNER JOIN lines.lines_list ll ON ll.line_id = p.line_id
		LEFT JOIN lines.label_templates lt ON lt.id = p.label_template_id
		WHERE p.id = $1`, id).Scan(
		&item.ID, &item.LineID, &item.LineName, &item.PrinterName, &item.Address,
		&item.LabelTemplateID, &item.LabelTemplateName, &item.PrintLanguage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return item, errors.New("printer v2 not found")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) syncPrintersV2IDSequence() error {
	_, err := r.store.db.Exec(`
		SELECT setval(
			pg_get_serial_sequence('lines.printers_v2', 'id'),
			COALESCE((SELECT MAX(id) FROM lines.printers_v2), 1)
		)`)
	return err
}

func (r *Repo) PrintersV2Add(lineID int, printerName, address string, labelTemplateID int, printLanguage string, userID int) (int, error) {
	var templateID any
	if labelTemplateID > 0 {
		templateID = labelTemplateID
	}

	if err := r.syncPrintersV2IDSequence(); err != nil {
		return 0, err
	}

	var id int
	err := r.store.db.QueryRow(`
		INSERT INTO lines.printers_v2 (line_id, printer_name, address, label_template_id, print_language, c_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		lineID, printerName, address, templateID, printLanguage, userID,
	).Scan(&id)
	return id, err
}

func (r *Repo) PrintersV2Delete(id int) error {
	res, err := r.store.db.Exec(`DELETE FROM lines.printers_v2 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("printer v2 not found")
	}
	return nil
}
