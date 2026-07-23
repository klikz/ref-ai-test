package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type LabBxRow struct {
	Serial     string `json:"serial"`
	Compressor string `json:"compressor"`
	BxModel    string `json:"bx_model"`
	LineNum    string `json:"line_num"`
	PointNum   string `json:"point_num"`
	StartTime  string `json:"start_time"`
	StopTime   string `json:"stop_time"`
	TestTime   string `json:"test_time"`
	BxResult   string `json:"bx_result"`
}

type LabInfoResponse struct {
	Serial string     `json:"serial"`
	Rows   []LabBxRow `json:"rows"`
}

func (r *Repo) LabInfoLookup(serial string) (LabInfoResponse, error) {
	serial = strings.TrimSpace(serial)
	resp := LabInfoResponse{Serial: serial, Rows: []LabBxRow{}}
	if serial == "" {
		return resp, errors.New("serial bo'sh")
	}

	labDB := r.store.labDB
	if labDB == nil {
		return resp, errors.New("Laboratoriya serveriga ulanib bo'lmadi")
	}

	timeoutSec := 10
	if v := strings.TrimSpace(os.Getenv("VTM_MSSQL_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	rows, err := labDB.QueryContext(ctx, `
SELECT
	CONVERT(varchar(100), BxNo) AS serial,
	CONVERT(varchar(200), Fd6) AS compressor,
	CONVERT(varchar(200), BxModel) AS BxModel,
	CONVERT(varchar(50), LineNum) AS LineNum,
	CONVERT(varchar(50), PointNum) AS PointNum,
	CONVERT(varchar(30), BeginDateTime, 120) AS StartTime,
	CONVERT(varchar(30), StopDateTime, 120) AS StopTime,
	CONVERT(varchar(50), TestTime) AS TestTime,
	CONVERT(varchar(100), BxResult) AS BxResult
FROM Vtm.dbo.BxData
WHERE BxNo = @p1
ORDER BY StopDateTime`,
		sql.Named("p1", serial),
	)
	if err != nil {
		return resp, errors.New("Laboratoriya serveriga ulanib bo'lmadi")
	}
	defer rows.Close()

	out := make([]LabBxRow, 0)
	for rows.Next() {
		var row LabBxRow
		var compressor, bxModel, lineNum, pointNum, startTime, stopTime, testTime, bxResult sql.NullString
		if err := rows.Scan(
			&row.Serial,
			&compressor,
			&bxModel,
			&lineNum,
			&pointNum,
			&startTime,
			&stopTime,
			&testTime,
			&bxResult,
		); err != nil {
			return resp, err
		}
		row.Compressor = compressor.String
		row.BxModel = bxModel.String
		row.LineNum = lineNum.String
		row.PointNum = pointNum.String
		row.StartTime = startTime.String
		row.StopTime = stopTime.String
		row.TestTime = testTime.String
		row.BxResult = bxResult.String
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return resp, err
	}

	resp.Rows = out
	return resp, nil
}

func (r *Repo) LabBxDataInsert(row LabBxRow, productSerial string, userID int) (int64, error) {
	productSerial = strings.TrimSpace(productSerial)
	if productSerial == "" {
		return 0, errors.New("product serial bo'sh")
	}
	bxResult := strings.TrimSpace(row.BxResult)
	if bxResult == "" {
		return 0, errors.New("bx_result bo'sh")
	}

	var id int64
	err := r.store.db.QueryRow(`
		INSERT INTO lines.lab_bx_data (
			product_serial, compressor, bx_model, line_num, point_num,
			start_time, stop_time, test_time, bx_result, c_user_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		productSerial,
		strings.TrimSpace(row.Compressor),
		strings.TrimSpace(row.BxModel),
		strings.TrimSpace(row.LineNum),
		strings.TrimSpace(row.PointNum),
		strings.TrimSpace(row.StartTime),
		strings.TrimSpace(row.StopTime),
		strings.TrimSpace(row.TestTime),
		bxResult,
		userID,
	).Scan(&id)
	return id, err
}

func (r *Repo) LabBxDataLatestBySerial(productSerial string) (LabBxRow, error) {
	productSerial = strings.TrimSpace(productSerial)
	out := LabBxRow{Serial: productSerial}
	if productSerial == "" {
		return out, errors.New("product serial bo'sh")
	}

	var compressor, bxModel, lineNum, pointNum, startTime, stopTime, testTime, bxResult sql.NullString
	err := r.store.db.QueryRow(`
		SELECT product_serial, compressor, bx_model, line_num, point_num,
			start_time, stop_time, test_time, bx_result
		FROM lines.lab_bx_data
		WHERE product_serial = $1
		ORDER BY c_time DESC, id DESC
		LIMIT 1`,
		productSerial,
	).Scan(
		&out.Serial,
		&compressor,
		&bxModel,
		&lineNum,
		&pointNum,
		&startTime,
		&stopTime,
		&testTime,
		&bxResult,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return LabBxRow{Serial: productSerial}, nil
	}
	if err != nil {
		return out, err
	}
	out.Compressor = compressor.String
	out.BxModel = bxModel.String
	out.LineNum = lineNum.String
	out.PointNum = pointNum.String
	out.StartTime = startTime.String
	out.StopTime = stopTime.String
	out.TestTime = testTime.String
	out.BxResult = bxResult.String
	return out, nil
}
