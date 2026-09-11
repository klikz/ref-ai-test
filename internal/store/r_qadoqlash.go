package store

import (
	"database/sql"
	"errors"
	"strings"
)

type QadoqlashSession struct {
	ID               int    `json:"id"`
	Serial           string `json:"serial"`
	AccSerial        string `json:"acc_serial"`
	DoorSerial       string `json:"door_serial"`
	FreezeDoorSerial string `json:"freeze_door_serial"`
	RefDoorSerial    string `json:"ref_door_serial"`
	Modeli           string `json:"modeli"`
	Time             string `json:"time"`
}

type QadoqlashLastScan struct {
	Serial           string `json:"serial"`
	AccSerial        string `json:"acc_serial"`
	DoorSerial       string `json:"door_serial"`
	FreezeDoorSerial string `json:"freeze_door_serial"`
	RefDoorSerial    string `json:"ref_door_serial"`
	Modeli           string `json:"modeli"`
	Compressor       string `json:"compressor"`
	BxModel          string `json:"bx_model"`
	BxResult         string `json:"bx_result"`
	Time             string `json:"time"`
	FilePath         string `json:"file_path,omitempty"`
	ProductID        int    `json:"product_id"`
}

type QadoqlashSessionsLastResponse struct {
	Sessions []QadoqlashSession  `json:"sessions"`
	LastScan *QadoqlashLastScan  `json:"last_scan,omitempty"`
}

func (r *Repo) QadoqlashSessionsGetLast(limit int) (QadoqlashSessionsLastResponse, error) {
	out := QadoqlashSessionsLastResponse{Sessions: []QadoqlashSession{}}
	if limit <= 0 {
		limit = LastRecordsLimit
	}
	if limit > 50 {
		limit = 50
	}

	rows, err := r.store.db.Query(`
		SELECT p.id,
			p.serial,
			COALESCE(NULLIF(pp.acc_serial, ''), NULLIF(p.acc_serial, ''), ''),
			COALESCE(NULLIF(pp.freeze_door_serial, ''), NULLIF(p.freeze_door_serial, ''), pp.door_serial, ''),
			COALESCE(NULLIF(pp.ref_door_serial, ''), NULLIF(p.ref_door_serial, ''), ''),
			COALESCE(m.modeli, ''),
			COALESCE(to_char(p.time, 'YYYY-MM-DD HH24:MI:SS'), '')
		FROM lines.products p
		LEFT JOIN lines.product_params pp ON pp.serial_number = p.serial
		LEFT JOIN production.models m ON m.id = COALESCE(NULLIF(pp.model_id, 0), p.model_id)
		WHERE p.line_id = $1
		  AND p.status = $2
		ORDER BY p.id DESC
		LIMIT $3`,
		QadoqlashLineID, ProductStatusActive, limit,
	)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	for rows.Next() {
		row := QadoqlashSession{}
		if err := rows.Scan(
			&row.ID, &row.Serial, &row.AccSerial,
			&row.FreezeDoorSerial, &row.RefDoorSerial,
			&row.Modeli, &row.Time,
		); err != nil {
			return out, err
		}
		row.DoorSerial = row.FreezeDoorSerial
		out.Sessions = append(out.Sessions, row)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	if len(out.Sessions) == 0 {
		return out, nil
	}

	latest := out.Sessions[0]
	last := &QadoqlashLastScan{
		Serial:           latest.Serial,
		AccSerial:        latest.AccSerial,
		DoorSerial:       latest.FreezeDoorSerial,
		FreezeDoorSerial: latest.FreezeDoorSerial,
		RefDoorSerial:    latest.RefDoorSerial,
		Modeli:           latest.Modeli,
		Time:             latest.Time,
		ProductID:        latest.ID,
	}

	lab, err := r.LabBxDataLatestBySerial(latest.Serial)
	if err != nil {
		return out, err
	}
	last.Compressor = strings.TrimSpace(lab.Compressor)
	last.BxModel = strings.TrimSpace(lab.BxModel)
	last.BxResult = strings.TrimSpace(lab.BxResult)

	photo, err := r.T3ScanPhotoLatestBySerial(latest.Serial)
	if err != nil {
		return out, err
	}
	if photo != nil {
		last.FilePath = strings.TrimSpace(photo.FilePath)
	} else {
		// packing photos may store sanitized serial; try product_id lookup
		photoByProduct, photoErr := r.packingScanPhotoLatestByProductID(latest.ID)
		if photoErr != nil {
			return out, photoErr
		}
		if photoByProduct != nil {
			last.FilePath = strings.TrimSpace(photoByProduct.FilePath)
		}
	}

	out.LastScan = last
	return out, nil
}

func (r *Repo) packingScanPhotoLatestByProductID(productID int) (*T3ScanPhoto, error) {
	if productID <= 0 {
		return nil, nil
	}
	row := T3ScanPhoto{}
	err := r.store.db.QueryRow(`
		SELECT id, COALESCE(product_id, 0), line_id, serial, file_path,
			to_char(captured_at, 'YYYY-MM-DD HH24:MI:SS'),
			COALESCE(user_id, 0)
		FROM lines.packing_scan_photos
		WHERE product_id = $1
		ORDER BY captured_at DESC
		LIMIT 1`,
		productID,
	).Scan(
		&row.ID, &row.ProductID, &row.LineID, &row.Serial, &row.FilePath,
		&row.CapturedAt, &row.UserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
