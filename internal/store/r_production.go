package store

import (
	"errors"
	"time"

	"github.com/klikz/api_v3/internal/models"
)

type ProductionReportInfo struct {
	Serial1   string `json:"serial_1"`
	Serial2   string `json:"serial_2"`
	ModelName string `json:"model_name"`
	Time      string `json:"time"`
}

// func (r *Repo) ProductionReportInfoByDate(startDate, endDate string) ([]ProductionReportInfo, error) {
// 	rows, err := r.store.db.Query(`
// 	select p.serial_1, p.serial_2, m.qisqa_nomi as model_name, to_char(p.c_time  , 'DD-MM-YYYY HH24:MI') "time"
// 	from production.product p, production.models m
// 	where p.c_time >= TO_DATE($1, 'YYYY-MM-DD')
// 	and p.c_time <= TO_DATE($2, 'YYYY-MM-DD') + interval '1 days'
// 	and m.id = p.model_id
// 	order by p.model_id
// 	`, startDate, endDate)

// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	allData := []ProductionReportInfo{}

// 	for rows.Next() {
// 		comp := ProductionReportInfo{}
// 		if err := rows.Scan(&comp.Serial1, &comp.Serial2, &comp.ModelName, &comp.Time); err != nil {
// 			return allData, err
// 		}
// 		allData = append(allData, comp)
// 	}
// 	if err = rows.Err(); err != nil {
// 		return allData, err
// 	}
// 	return allData, nil
// }

func (r *Repo) ProductCount(startDate, endDate string) (int, error) {
	count := 0
	err := r.store.db.QueryRow(`
	select count(*)
	from lines.products p
	where p."time" between to_timestamp((case when $1 in('') then to_char(now(), 'YYYY-MM-DD') else $1 end), 'YYYY-MM-DD')
	and to_timestamp((case when $2 in('') then (to_char(now(), 'YYYY-MM-DD')) else $2 end), 'YYYY-MM-DD') + interval '1 days'
	`, startDate, endDate).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repo) ProductSerialCheck(serial string) (bool, error) {
	id := 0
	err := r.store.db.QueryRow(`
	select p.id
		from lines.products p 
		where p.serial = $1
	`, serial).Scan(&id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, errors.New("serial xato")
		}
		return false, err
	}
	if id > 0 {
		return true, nil
	}
	return false, nil
}

// func (r *Repo) ProductCountByDate(startDate, endDate string) (int, error) {
// 	count := 0
// 	err := r.store.db.QueryRow(`
// 	select  count(*)
// 	from production.product p
// 	where p.c_time >= TO_DATE($1, 'YYYY-MM-DD')
// 	and p.c_time <= TO_DATE($2, 'YYYY-MM-DD') + interval '1 days'
// 	`, startDate, endDate).Scan(&count)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return count, nil
// }

func (r *Repo) ProductionCountModels(startDate, endDate string, lineIDs, modelIDs []int) ([]models.ModelsCount, error) {
	return r.productionCountModelsQuery(startDate, endDate, lineIDs, modelIDs)
}

func (r *Repo) ProductionCountModelsByLines(startDate, endDate string, lineIDs []int, modelIDs []int) ([]models.ModelsCount, error) {
	return r.productionCountModelsQuery(startDate, endDate, lineIDs, modelIDs)
}

func (r *Repo) ProductionCountModelsByTimeRange(start, end time.Time, lineIDs, modelIDs []int) ([]models.ModelsCount, error) {
	if end.Before(start) || end.Equal(start) {
		return []models.ModelsCount{}, nil
	}
	from := start.Format("2006-01-02 15:04:05")
	to := end.Format("2006-01-02 15:04:05")
	return r.productionCountModelsQueryRaw(from, to, lineIDs, modelIDs)
}

func (r *Repo) productionCountModelsQueryRaw(from, to string, lineIDs, modelIDs []int) ([]models.ModelsCount, error) {
	query := `
	select p.line_id, ll."name", m.id, m.modeli, COALESCE(m.seriya_raqami, ''), COALESCE(m.odoo_code, ''), count(*)
	from lines.products p
	inner join production.models m on p.model_id = m.id
	inner join lines.lines_list ll on ll.line_id = p.line_id
	where p."time" >= $1::timestamp
	AND p."time" < $2::timestamp
	AND (cardinality($3::int[]) = 0 OR p.line_id = ANY($3))
	AND (cardinality($4::int[]) = 0 OR m.id = ANY($4))
	group by p.line_id, m.id, m.modeli, m.seriya_raqami, m.odoo_code, ll."name" order by ll."name", m.modeli`

	rows, err := r.store.db.Query(query, from, to, intSliceParam(lineIDs), intSliceParam(modelIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.ModelsCount{}
	for rows.Next() {
		comp := models.ModelsCount{}
		if err := rows.Scan(&comp.LineID, &comp.LineName, &comp.ModelID, &comp.ModelName, &comp.SeriyaRaqami, &comp.OdooCode, &comp.Count); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	return allData, rows.Err()
}

func (r *Repo) productionCountModelsQuery(startDate, endDate string, lineIDs, modelIDs []int) ([]models.ModelsCount, error) {
	from, to, err := NormalizeReportTimeRange(startDate, endDate)
	if err != nil {
		return nil, err
	}
	return r.productionCountModelsQueryRaw(from, to, lineIDs, modelIDs)
}

type ReportWithSerial struct {
	ID           int    `json:"id"`
	Serial1      string `json:"serial_1"`
	Serial2      string `json:"serial_2"`
	ModelName    string `json:"model_name"`
	SeriyaRaqami string `json:"seriya_raqami"`
	ModelID      int    `json:"model_id"`
	Time         string `json:"time"`
	GsCode       string `json:"gs_code"`
	GsID         int    `json:"gs_id"`
}

func (r *Repo) ProductionReportWithSerials(startDate, endDate string) ([]ReportWithSerial, error) {
	orderBy := "m.modeli"
	if startDate == "" {
		orderBy = `p."time" desc, m.modeli`
	}

	query := `
		select p.id,
			COALESCE(p.serial, ''),
			COALESCE(p.acc_serial, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.seriya_raqami, ''),
			COALESCE(p.model_id, 0),
			to_char(p."time", 'YYYY-MM-DD HH24-MI') as time,
			case when g."data" is not null then g."data" else ' ' end as gs_data,
			case when g.id is not null then g.id else 0 end as gs_id
		FROM lines.products p
		left join production.models m on m.id = p.model_id
		left join production.gscodes g on g.product_id = p.id
		where p."time" between to_timestamp((case when $1 in('') then to_char(now(), 'YYYY-MM-DD') else $1 end), 'YYYY-MM-DD')
		and to_timestamp((case when $2 in('') then (to_char(now(), 'YYYY-MM-DD')) else $2 end), 'YYYY-MM-DD') + interval '1 days'
		order by ` + orderBy

	rows, err := r.store.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []ReportWithSerial{}

	for rows.Next() {
		comp := ReportWithSerial{}
		if err := rows.Scan(&comp.ID, &comp.Serial1, &comp.Serial2, &comp.ModelName, &comp.SeriyaRaqami, &comp.ModelID, &comp.Time, &comp.GsCode, &comp.GsID); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}
