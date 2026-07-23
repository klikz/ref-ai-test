package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/klikz/api_v3/internal/models"
	"github.com/lib/pq"
)

type LinesGPComponent struct {
	ID          int    `json:"id"`
	ComponentID int    `json:"component_id"`
	MainCode    string `json:"main_code"`
	NameShortUz string `json:"name_short_uz"`
	LineID      int    `json:"line_id"`
	LineName    string `json:"line_name"`
	Index_1     string `json:"index_1"`
	Index_2     string `json:"index_2"`
	Serial      string `json:"serial"`
	Count       int    `json:"count"`
}

func (r *Repo) LinesGetGPComponents() ([]LinesGPComponent, error) {
	rows, err := r.store.db.Query(`select gc.id, c.id, c.manufacturer_code, c.standard_name_uz, gc.line_id, ll."name" as line, gc.index_1, gc.index_2, gc.serial 
		from lines.gp_components gc, production.components c, lines.lines_list ll 
		where gc.component_id = c.id 
		and ll.line_id = gc.line_id 
		order by gc.line_id, gc.component_id `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []LinesGPComponent{}

	for rows.Next() {
		comp := LinesGPComponent{}
		if err := rows.Scan(&comp.ID, &comp.ComponentID, &comp.MainCode, &comp.NameShortUz, &comp.LineID, &comp.LineName, &comp.Index_1, &comp.Index_2, &comp.Serial); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) LinesGetGPComponentById(id int) (LinesGPComponent, error) {

	data := LinesGPComponent{}
	err := r.store.db.QueryRow(`select gc.id, c.id, c.manufacturer_code, c.standard_name_uz, gc.line_id, ll."name" as line, gc.index_1, gc.index_2, gc.serial, gc.count   
		from lines.gp_components gc, production.components c, lines.lines_list ll 
		where gc.component_id = c.id 
		and ll.line_id = gc.line_id 
		and gc.id = $1`, id).Scan(&data.ID, &data.ComponentID, &data.MainCode, &data.NameShortUz, &data.LineID, &data.LineName, &data.Index_1, &data.Index_2, &data.Serial, &data.Count)
	if err != nil {
		return data, err
	}
	return data, nil

}

func (r *Repo) LinesAddGpComponent(component_id, line_id, user_id int, index_1, index_2 string, serial string) error {
	_, err := r.store.db.Exec(`
		INSERT INTO lines.gp_components
		(component_id, line_id, user_id, index_1, index_2, serial)
		VALUES($1, $2, $3, $4, $5, $6);
		`, component_id, line_id, user_id, index_1, index_2, serial)
	if err != nil {
		if strings.Contains(err.Error(), "gp_components_un") {
			return errors.New("gp_component kiritilgan")
		}
		return err
	}
	return nil
}

type Lines struct {
	LineID int    `json:"line_id"`
	Name   string `json:"name"`
}

func (r *Repo) LinesGetAll() ([]Lines, error) {
	rows, err := r.store.db.Query(`select ll.line_id, ll."name" from lines.lines_list ll 
		where ll.status = true
		order by ll."name" `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []Lines{}

	for rows.Next() {
		comp := Lines{}
		if err := rows.Scan(&comp.LineID, &comp.Name); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) LinesGpComponentDelete(line_id, component_id int) error {
	_, err := r.store.db.Exec(`
		delete from lines.gp_components 
		where line_id  = $1
		and component_id = $2
	`, line_id, component_id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) LinesGetProductIdBySerial(line_id int, serial string) (int, error) {

	id := 0
	err := r.store.db.QueryRow(`
		select p.id
		from lines.products p 
		where p.line_id = $1
		and p.serial = $2
	`, line_id, serial).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repo) LinesGetModelIDByProductID(productID int) (int, error) {
	var modelID int
	err := r.store.db.QueryRow(`
		SELECT COALESCE(model_id, 0)
		FROM lines.products
		WHERE id = $1`, productID).Scan(&modelID)
	if err != nil {
		return 0, err
	}
	return modelID, nil
}

func (r *Repo) ProductAccSerialBySerial(serial string) (string, error) {
	var accSerial string
	err := r.store.db.QueryRow(`
		SELECT COALESCE(acc_serial, '')
		FROM lines.products
		WHERE serial = $1
		ORDER BY id DESC
		LIMIT 1
	`, strings.TrimSpace(serial)).Scan(&accSerial)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(accSerial), nil
}

func (r *Repo) linesProductDuplicateCheck(line_id int, serial, acc_serial string) error {
	serial = strings.TrimSpace(serial)
	acc_serial = strings.TrimSpace(acc_serial)

	if acc_serial != "" {
		var existingID int
		err := r.store.db.QueryRow(
			`SELECT id FROM lines.products WHERE acc_serial = $1 LIMIT 1`,
			acc_serial,
		).Scan(&existingID)
		if err == nil {
			return errors.New("bu aksessuar nomer allaqachon kiritilgan")
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	if serial != "" {
		var existingID int
		err := r.store.db.QueryRow(
			`SELECT id FROM lines.products WHERE line_id = $1 AND serial = $2 LIMIT 1`,
			line_id, serial,
		).Scan(&existingID)
		if err == nil {
			return errors.New("bu serial nomer ushbu liniyada allaqachon kiritilgan")
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	return nil
}

func linesProductDuplicateError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "повторяющееся значение") {
			return errors.New("bunday ma'lumot allaqachon kiritilgan")
		}
		return nil
	}

	switch pqErr.Constraint {
	case "products_unique":
		return errors.New("bu aksessuar nomer allaqachon kiritilgan")
	case "products_un_2":
		return errors.New("bu serial nomer ushbu liniyada allaqachon kiritilgan")
	}

	if strings.Contains(strings.ToLower(pqErr.Detail), "(acc_serial)") {
		return errors.New("bu aksessuar nomer allaqachon kiritilgan")
	}
	if strings.Contains(strings.ToLower(pqErr.Detail), "(serial)") {
		return errors.New("bu serial nomer ushbu liniyada allaqachon kiritilgan")
	}

	return errors.New("bunday ma'lumot allaqachon kiritilgan")
}

func (r *Repo) linesAddProductTx(tx *sql.Tx, line_id, component_id, user_id, model_id int, serial, acc_serial string) (int, error) {
	serial = strings.TrimSpace(serial)
	acc_serial = strings.TrimSpace(acc_serial)

	id := 0
	err := tx.QueryRow(`
		INSERT INTO lines.products (line_id, component_id, serial, user_id, model_id, acc_serial, status)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
		ON CONFLICT ON CONSTRAINT products_unique DO NOTHING
		RETURNING id`, line_id, component_id, serial, user_id, model_id, acc_serial, ProductStatusActive).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("bu aksessuar nomer allaqachon kiritilgan")
		}
		if dupErr := linesProductDuplicateError(err); dupErr != nil {
			return 0, dupErr
		}
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("bu aksessuar nomer allaqachon kiritilgan")
	}
	return id, nil
}

func (r *Repo) LinesAddProduct(line_id, component_id, user_id, model_id int, serial, acc_serial string) (int, error) {
	serial = strings.TrimSpace(serial)
	acc_serial = strings.TrimSpace(acc_serial)

	if err := r.linesProductDuplicateCheck(line_id, serial, acc_serial); err != nil {
		return 0, err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	id, err := r.linesAddProductTx(tx, line_id, component_id, user_id, model_id, serial, acc_serial)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repo) LinesDeleteProduct(id int) error {
	_, err := r.store.db.Exec(`
		delete from lines.products 
		where id = $1`, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Repo) LinesIncreaseCount(gp_id int) (int, error) {
	var count int
	err := s.store.db.QueryRow(`
		UPDATE lines.gp_components t
		SET count = case when t.pr_date=now()::date then t.count+1 else 1 end,
			pr_date =  case when t.pr_date=now()::date then t.pr_date else now() end
		WHERE t.id  = $1
		returning t.count 
	`, gp_id).Scan(&count)
	if err != nil {
		return count, err
	}
	return count, nil
}

func (s *Repo) LinesDecreaseCount(line_id, component_id int) (int, error) {
	var count int
	err := s.store.db.QueryRow(`
		update lines.gp_components 
		set count = count -1
		where line_id = $1
		and component_id = $2
		returning count 
	`, line_id, component_id).Scan(&count)
	if err != nil {
		return count, err
	}
	return count, nil
}

type LinesLastComponents struct {
	ID           int    `json:"id"`
	LineName     string `json:"line_name"`
	MainCode     string `json:"main_code"`
	Serial       string `json:"serial"`
	SeriyaRaqami string `json:"seriya_raqami"`
	Modeli       string `json:"modeli"`
	Index_1      string `json:"index_1"`
	Index_2      string `json:"index_2"`
	Time         string `json:"time"`
}

func (s *Repo) LinesGetLastComponents(line_id int) ([]LinesLastComponents, error) {
	rows, err := s.store.db.Query(`select p.id, ll."name" as line_name, c.manufacturer_code, p.serial, gc.serial, gc.index_1, gc.index_2, to_char(p."time", 'DD-MM-YYYY HH24-MI') as "time" 
		from lines.products p, lines.lines_list ll, lines.gp_components gc, production.components c 
		where p.line_id = ll.line_id 
		and p.line_id = $1
		and gc.component_id = p.component_id 
		and c.id = gc.component_id 
		order by p.id desc 
		limit $2`, line_id, LastRecordsLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []LinesLastComponents{}

	for rows.Next() {
		comp := LinesLastComponents{}
		if err := rows.Scan(&comp.ID, &comp.LineName, &comp.MainCode, &comp.Serial, &comp.SeriyaRaqami, &comp.Index_1, &comp.Index_2, &comp.Time); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (s *Repo) LinesGetT1Last() ([]models.LReport, error) {
	rows, err := s.store.db.Query(`select tp.id, tp.serial, m.qisqa_nomi, m.modeli, to_char(tp."time", 'YYYY-MM-DD HH24:MI' )
									from lines.t1_printed tp, production.models m 
									where m.id = tp.model_id 
									order by id desc limit $1`, LastRecordsLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.LReport{}

	for rows.Next() {
		comp := models.LReport{}
		if err := rows.Scan(&comp.Id, &comp.Serial, &comp.Model, &comp.ModelNomi, &comp.Time); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (s *Repo) LinesGetLast(line_id int) ([]models.LReport, error) {
	rows, err := s.store.db.Query(`select p.id, p.serial, COALESCE(p.acc_serial, ''), m.qisqa_nomi, m.modeli, to_char(p."time", 'YYYY-MM-DD HH24-MI')
		from lines.products p, production.models m 
		where p.line_id = $1
		and p.model_id = m.id
		order by p.id desc limit $2`, line_id, LastRecordsLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.LReport{}

	for rows.Next() {
		comp := models.LReport{}
		if err := rows.Scan(&comp.Id, &comp.Serial, &comp.AccSerial, &comp.Model, &comp.ModelNomi, &comp.Time); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (s *Repo) LinesGetLastSerialByLineModel(lineID, modelID int) (string, error) {
	serial := ""
	err := s.store.db.QueryRow(`
		SELECT p.serial
		FROM lines.products p
		WHERE p.line_id = $1
		  AND p.model_id = $2
		ORDER BY p.id DESC
		LIMIT 1
	`, lineID, modelID).Scan(&serial)
	if err != nil {
		return "", err
	}
	return serial, nil
}

const linesReportSelect = `SELECT 
		p.id,
		p.serial,
		m.id,
		m.modeli,
		m.qisqa_nomi,
		m.gs1_ean13,
		m.odoo_code,
		ll.line_id AS line_id,
		ll.name,
		COALESCE(g.data, '') AS data,
		to_char(p.time, 'YYYY-MM-DD HH24:MI') AS time
		FROM lines.products p
		JOIN production.models m ON p.model_id = m.id
		JOIN lines.lines_list ll ON ll.line_id = p.line_id
		LEFT JOIN production.gscodes g ON g.product_id = p.id 
		WHERE
			p.time >= $1::timestamp
		AND p.time < $2::timestamp
		AND (cardinality($3::int[]) = 0 OR p.line_id = ANY($3))
		AND (cardinality($4::int[]) = 0 OR m.id = ANY($4))`

func (s *Repo) LinesReportCount(lineIDs, modelIDs []int, date, date2 string) (int, error) {
	from, to, err := NormalizeReportTimeRange(date, date2)
	if err != nil {
		return 0, err
	}
	var count int
	err = s.store.db.QueryRow(
		`SELECT COUNT(*) FROM (`+linesReportSelect+`) AS report_rows`,
		from, to, intSliceParam(lineIDs), intSliceParam(modelIDs),
	).Scan(&count)
	return count, err
}

func (s *Repo) LinesReport(lineIDs, modelIDs []int, date, date2 string, limit, offset int) ([]models.LReport, error) {
	from, to, err := NormalizeReportTimeRange(date, date2)
	if err != nil {
		return nil, err
	}
	query := linesReportSelect + ` ORDER BY ll.name, m.modeli, p.id`
	args := []any{from, to, intSliceParam(lineIDs), intSliceParam(modelIDs)}
	if limit > 0 {
		query += ` LIMIT $5 OFFSET $6`
		args = append(args, limit, offset)
	}
	rows, err := s.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.LReport{}

	for rows.Next() {
		comp := models.LReport{}
		if err := rows.Scan(&comp.Id, &comp.Serial, &comp.ModelId, &comp.Model, &comp.ModelNomi, &comp.Gs1, &comp.OdooCode,
			&comp.LineID, &comp.LineName, &comp.GsCode, &comp.Time); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

// func (s *Repo) LinesReport(line_id, model_id int, date, date2 string) ([]models.LReport, error) {
// 	rows, err := s.store.db.Query(`SELECT
// 									p.id,
// 									p.serial,
// 									m.id,
// 									m.modeli,
// 									m.qisqa_nomi,
// 									m.odoo_code,
// 									ll.line_id,
// 									ll.name,
// 									to_char(p.time, 'YYYY-MM-DD HH24:MI') AS time
// 								FROM lines.products p
// 								JOIN production.models m
// 									ON p.model_id = m.id
// 								JOIN lines.lines_list ll
// 									ON ll.line_id = p.line_id
// 								WHERE
// 									p.time >= COALESCE(
// 										NULLIF($1, '')::date,
// 										CURRENT_DATE
// 									)
// 									AND p.time < COALESCE(
// 										NULLIF($2, '')::date,
// 										CURRENT_DATE
// 									) + INTERVAL '1 day'
// 									AND (
// 										NULLIF($3, 0) IS NULL
// 										OR p.line_id = $3
// 									)
// 									AND (
// 										NULLIF($4, 0) IS NULL
// 										OR m.id = $4
// 									)
// 									order by ll.name, m.modeli  `, date, date2, line_id, model_id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	allData := []models.LReport{}

// 	for rows.Next() {
// 		comp := models.LReport{}
// 		if err := rows.Scan(&comp.Id, &comp.Serial, &comp.ModelId, &comp.Model, &comp.ModelNomi, &comp.OdooCode,
// 			&comp.LineID, &comp.LineName, &comp.Time); err != nil {
// 			return allData, err
// 		}
// 		allData = append(allData, comp)
// 	}
// 	if err = rows.Err(); err != nil {
// 		return allData, err
// 	}
// 	return allData, nil
// }

type LinePrinters struct {
	ID          int    `json:"id"`
	LineId      int    `json:"line_id"`
	LineName    string `json:"line_name"`
	Address     string `json:"address"`
	PrinterName string `json:"printer_name"`
	FolderName  string `json:"folder_name"`
}

func (s *Repo) LinesGetPrinters() ([]LinePrinters, error) {
	rows, err := s.store.db.Query(`select p.id, p.line_id, ll."name", p.address, p.printer_name, ll.folder_name  
		from lines.printers p, lines.lines_list ll 
		where ll.line_id = p.line_id 
		order by p.line_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []LinePrinters{}

	for rows.Next() {
		comp := LinePrinters{}
		if err := rows.Scan(&comp.ID, &comp.LineId, &comp.LineName, &comp.Address, &comp.PrinterName, &comp.FolderName); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil

}

func (r *Repo) PrinterInfoById(printer_id int) (LinePrinters, error) {
	info := LinePrinters{}
	err := r.store.db.QueryRow(`select p.id, p.line_id, ll."name", p.address, p.printer_name, ll.folder_name  
		from lines.printers p, lines.lines_list ll 
		where ll.line_id = p.line_id 
		and p.id = $1`, printer_id).Scan(&info.ID, &info.LineId, &info.LineName, &info.Address, &info.PrinterName, &info.FolderName)

	if err != nil {
		return info, err
	}

	return info, nil
}

// func (r *Repo) LinesGetFolderName(line_id int) (string, error) {
// 	name := ""

// 	err := r.store.db.QueryRow(`
// 	select fn.folder_name  from lines.folder_names fn
// where fn.line_id = $1`, line_id).Scan(&name)

// 	if err != nil {
// 		return name, errors.New("folder_name not found")
// 	}
// 	return name, nil
// }

const FinPressLineID = 8
const RadiatorLineID = 9
const KlapanLineID = 10
const EshikLineID = 11
const QadoqlashLineID = 12

func (r *Repo) LinesEshikLineID() (int, error) {
	var ok int
	err := r.store.db.QueryRow(`
		SELECT 1
		FROM lines.lines_list
		WHERE line_id = $1 AND status = true`,
		EshikLineID,
	).Scan(&ok)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("eshik liniyasi topilmadi (line_id=11)")
		}
		return 0, err
	}
	return EshikLineID, nil
}

func (r *Repo) LinesFinPressLineID() (int, error) {
	var ok int
	err := r.store.db.QueryRow(`
		SELECT 1
		FROM lines.lines_list
		WHERE line_id = $1 AND status = true`,
		FinPressLineID,
	).Scan(&ok)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("fin press liniyasi topilmadi (id=8)")
		}
		return 0, err
	}
	return FinPressLineID, nil
}

func (r *Repo) LinesRadiatorLineID() (int, error) {
	var ok int
	err := r.store.db.QueryRow(`
		SELECT 1
		FROM lines.lines_list
		WHERE line_id = $1 AND status = true`,
		RadiatorLineID,
	).Scan(&ok)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("radiator liniyasi topilmadi (id=9)")
		}
		return 0, err
	}
	return RadiatorLineID, nil
}

func (r *Repo) LinesKlapanLineID() (int, error) {
	var ok int
	err := r.store.db.QueryRow(`
		SELECT 1
		FROM lines.lines_list
		WHERE line_id = $1 AND status = true`,
		KlapanLineID,
	).Scan(&ok)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("klapan yig'ish liniyasi topilmadi (id=10)")
		}
		return 0, err
	}
	return KlapanLineID, nil
}

func (r *Repo) LinesBalanceUpdate(quantity float64, line_id, component_id, user_id int, source, comment string) error {
	_, err := r.LinesBalanceApplyChange(BalanceChangeParams{
		LineID:         line_id,
		ComponentID:    component_id,
		QuantityChange: quantity,
		UserID:         user_id,
		Source:         source,
		Comment:        comment,
	})
	return err
}

type LinesBalance struct {
	ID          int     `json:"id"`
	MainCode    string  `json:"main_code"`
	OdooCode    string  `json:"odoo_code"`
	ComponentId int     `json:"component_id"`
	FullNameUz  string  `json:"full_name_uz"`
	Quantity    float64 `json:"quantity"`
}

func (r *Repo) LinesGetBalance(line_id int) ([]LinesBalance, error) {
	rows, err := r.store.db.Query(`
	SELECT b.id,
		COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''),
		COALESCE(c.odoo_code, ''),
		c.id AS component_id,
		COALESCE(c.full_name_uz, ''),
		b.quantity
	FROM lines.balance b
	INNER JOIN production.components c ON c.id = b.component_id
	WHERE b.line_id = $1
	ORDER BY COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, '')`, line_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []LinesBalance{}

	for rows.Next() {
		comp := LinesBalance{}
		if err := rows.Scan(&comp.ID, &comp.MainCode, &comp.OdooCode, &comp.ComponentId, &comp.FullNameUz, &comp.Quantity); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) LinesT1InsertProduct(serial string, model_id int) error {
	_, err := r.store.db.Exec(`
		INSERT INTO lines.t1_printed (serial, model_id) VALUES ($1, $2)
	`, serial, model_id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Repo) LinesRejaGet() (int, error) {
	count := 0

	err := s.store.db.QueryRow(`select r.count  from lines.reja r limit 1`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil

}

func (s *Repo) LinesRejaUpdate(count int) error {
	_, err := s.store.db.Exec(`update lines.reja set count = $1`, count)
	if err != nil {
		return err
	}
	return nil

}
