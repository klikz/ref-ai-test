package store

import (
	"database/sql"
	"errors"
	"strings"
)

type SerialInfoResponse struct {
	Serial     string                 `json:"serial"`
	InProducts bool                   `json:"in_products"`
	Catalog    *SerialInfoCatalog     `json:"catalog,omitempty"`
	Product    *SerialInfoProduct     `json:"product,omitempty"`
	Compressor *SerialInfoCompressor  `json:"compressor,omitempty"`
	ScanPhoto  *T3ScanPhoto           `json:"scan_photo,omitempty"`
	Lab        *SerialInfoLab         `json:"lab,omitempty"`
}

type SerialInfoLab struct {
	Source string     `json:"source"` // mssql | postgres
	Latest *LabBxRow  `json:"latest,omitempty"`
	Rows   []LabBxRow `json:"rows,omitempty"`
}

type SerialInfoCatalog struct {
	ModelID            int    `json:"model_id"`
	Artikul            string `json:"artikul"`
	Modeli             string `json:"modeli"`
	ModelNomi          string `json:"model_nomi"`
	ImportCode         string `json:"import_code"`
	SeriyaRaqami       string `json:"seriya_raqami"`
	AccSerialRule      string `json:"acc_serial_rule"`
	CompressorSerialRule string `json:"compressor_serial_rule"`
	Gs1Shablon         string `json:"gs1_shablon"`
}

type SerialInfoProduct struct {
	ProductID     int    `json:"product_id"`
	AccSerial     string `json:"acc_serial"`
	Status        string `json:"status"`
	LineID        int    `json:"line_id"`
	LineName      string `json:"line_name"`
	RegisteredAt  string `json:"registered_at"`
	TransferredAt string `json:"transferred_at"`
	UserName      string `json:"user_name"`
	GsCode        string `json:"gs_code"`
	GsCodeAt      string `json:"gs_code_at"`
	OdooCode      string `json:"odoo_code"`
	Brend         string `json:"brend"`
	Gs1EAN13      string `json:"gs1_ean13"`
	Modeli        string `json:"modeli"`
	ModelNomi     string `json:"model_nomi"`
}

type SerialInfoCompressor struct {
	ParamsID         int64  `json:"params_id"`
	SerialNumber     string `json:"serial_number"`
	CompressorSerial string `json:"compressor_serial"`
	AccSerial        string `json:"acc_serial"`
	DoorSerial       string `json:"door_serial"`
	FreezeDoorSerial string `json:"freeze_door_serial"`
	RefDoorSerial    string `json:"ref_door_serial"`
	GsCode           string `json:"gscode"`
	ModelID          int    `json:"model_id"`
	Modeli           string `json:"modeli"`
	CreatedAt        string `json:"created_at"`
	UserName         string `json:"user_name"`
	MatchedBy        string `json:"matched_by"`
}

func (r *Repo) SerialInfoLookup(serial string) (SerialInfoResponse, error) {
	serial = strings.TrimSpace(serial)
	resp := SerialInfoResponse{Serial: serial}
	if serial == "" {
		return resp, errors.New("serial bo'sh")
	}

	params, matchedBy, err := r.serialInfoResolveParams(serial)
	if err != nil {
		return resp, err
	}
	productSerial := serial
	if params.ID > 0 {
		resp.Compressor = &SerialInfoCompressor{
			ParamsID:         params.ID,
			SerialNumber:     params.SerialNumber,
			CompressorSerial: params.CompressorSerial,
			AccSerial:        params.AccSerial,
			DoorSerial:       params.DoorSerial,
			FreezeDoorSerial: params.FreezeDoorSerial,
			RefDoorSerial:    params.RefDoorSerial,
			GsCode:           params.GsCode,
			ModelID:          params.ModelID,
			Modeli:           params.Modeli,
			CreatedAt:        params.CreatedAt,
			UserName:         params.UserName,
			MatchedBy:        matchedBy,
		}
		if matchedBy == "compressor" && params.SerialNumber != "" {
			productSerial = params.SerialNumber
		}
	}

	catalogSerial := productSerial
	if len(catalogSerial) >= 7 {
		short, shortErr := r.ModelsShortInfoBySerial(catalogSerial)
		if shortErr == nil {
			resp.Catalog = &SerialInfoCatalog{
				ModelID:              short.ModelId,
				Artikul:              short.ArtikulRaqami,
				Modeli:               short.ModelName,
				ImportCode:           short.ImportCode,
				SeriyaRaqami:         short.SeriyaRaqami,
				AccSerialRule:        short.AccSerial,
				CompressorSerialRule: short.CompressorSerial,
				Gs1Shablon:           short.Gs1Shablon,
			}
		}
	}

	product, err := r.serialInfoLoadProduct(productSerial)
	if err != nil {
		return resp, err
	}
	if product == nil && productSerial != serial {
		product, err = r.serialInfoLoadProduct(serial)
		if err != nil {
			return resp, err
		}
	}
	if product != nil {
		resp.InProducts = true
		resp.Product = product
		if resp.Catalog == nil {
			resp.Catalog = &SerialInfoCatalog{
				Modeli:    product.Modeli,
				ModelNomi: product.ModelNomi,
			}
		} else if resp.Catalog.ModelNomi == "" {
			resp.Catalog.ModelNomi = product.ModelNomi
		}
	}

	if resp.Catalog != nil && resp.Catalog.CompressorSerialRule == "" && resp.Catalog.ModelID > 0 {
		var rule string
		if err := r.store.db.QueryRow(`
			SELECT COALESCE(compressor_serial, '')
			FROM production.models
			WHERE id = $1`, resp.Catalog.ModelID).Scan(&rule); err == nil {
			resp.Catalog.CompressorSerialRule = rule
		}
	} else if resp.Catalog == nil && params.ModelID > 0 {
		var rule, modeli, artikul, seriya, acc string
		if err := r.store.db.QueryRow(`
			SELECT COALESCE(modeli, ''), COALESCE(qisqa_nomi, ''), COALESCE(seriya_raqami, ''),
				COALESCE(acc_serial, ''), COALESCE(compressor_serial, '')
			FROM production.models
			WHERE id = $1`, params.ModelID).Scan(&modeli, &artikul, &seriya, &acc, &rule); err == nil {
			resp.Catalog = &SerialInfoCatalog{
				ModelID:              params.ModelID,
				Artikul:              artikul,
				Modeli:               modeli,
				SeriyaRaqami:         seriya,
				AccSerialRule:        acc,
				CompressorSerialRule: rule,
			}
		}
	}

	photoSerials := []string{serial, productSerial}
	for _, photoSerial := range photoSerials {
		if photoSerial == "" {
			continue
		}
		photo, photoErr := r.T3ScanPhotoLatestBySerial(photoSerial)
		if photoErr != nil {
			return resp, photoErr
		}
		if photo == nil {
			photo, photoErr = r.T3ScanPhotoLatestBySerial(sanitizeT3ScanSerial(photoSerial))
			if photoErr != nil {
				return resp, photoErr
			}
		}
		if photo != nil {
			resp.ScanPhoto = photo
			break
		}
	}

	resp.Lab = r.serialInfoLoadLab(productSerial, serial)

	if !resp.InProducts && resp.Catalog == nil && resp.ScanPhoto == nil && resp.Compressor == nil && resp.Lab == nil {
		return resp, errors.New("serial bo'yicha ma'lumot topilmadi")
	}

	return resp, nil
}

func (r *Repo) serialInfoLoadLab(serials ...string) *SerialInfoLab {
	seen := map[string]struct{}{}
	for _, serial := range serials {
		serial = strings.TrimSpace(serial)
		if serial == "" {
			continue
		}
		if _, ok := seen[serial]; ok {
			continue
		}
		seen[serial] = struct{}{}

		labResp, err := r.LabInfoLookup(serial)
		if err == nil && len(labResp.Rows) > 0 {
			latest := labResp.Rows[len(labResp.Rows)-1]
			return &SerialInfoLab{
				Source: "mssql",
				Latest: &latest,
				Rows:   labResp.Rows,
			}
		}

		pgRow, err := r.LabBxDataLatestBySerial(serial)
		if err == nil && strings.TrimSpace(pgRow.BxResult) != "" {
			row := pgRow
			return &SerialInfoLab{
				Source: "postgres",
				Latest: &row,
				Rows:   []LabBxRow{row},
			}
		}
	}
	return nil
}

func (r *Repo) serialInfoResolveParams(serial string) (ProductParams, string, error) {
	params, err := r.ProductParamsGetBySerial(serial)
	if err != nil {
		return ProductParams{}, "", err
	}
	if params.ID > 0 {
		return params, "serial", nil
	}

	params, err = r.ProductParamsGetByCompressor(serial)
	if err != nil {
		return ProductParams{}, "", err
	}
	if params.ID > 0 {
		return params, "compressor", nil
	}
	return ProductParams{}, "", nil
}

func (r *Repo) serialInfoLoadProduct(serial string) (*SerialInfoProduct, error) {
	product := SerialInfoProduct{}
	err := r.store.db.QueryRow(`
		SELECT p.id,
			COALESCE(p.acc_serial, ''),
			COALESCE(p.status, ''),
			p.line_id,
			COALESCE(ll.name, ''),
			COALESCE(to_char(p.time, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(to_char(p.transferred_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			COALESCE((
				SELECT g.data
				FROM production.gscodes g
				WHERE g.product_id = p.id
				ORDER BY g.u_time DESC NULLS LAST, g.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT to_char(g.u_time, 'YYYY-MM-DD HH24:MI:SS')
				FROM production.gscodes g
				WHERE g.product_id = p.id
				ORDER BY g.u_time DESC NULLS LAST, g.id DESC
				LIMIT 1
			), ''),
			COALESCE(m.odoo_code, ''),
			COALESCE(m.brend, ''),
			COALESCE(m.gs1_ean13, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.qisqa_nomi, '')
		FROM lines.products p
		JOIN production.models m ON m.id = p.model_id
		JOIN lines.lines_list ll ON ll.line_id = p.line_id
		LEFT JOIN auth.users u ON u.id = p.user_id
		WHERE p.serial = $1
		ORDER BY p.id DESC
		LIMIT 1`,
		serial,
	).Scan(
		&product.ProductID,
		&product.AccSerial,
		&product.Status,
		&product.LineID,
		&product.LineName,
		&product.RegisteredAt,
		&product.TransferredAt,
		&product.UserName,
		&product.GsCode,
		&product.GsCodeAt,
		&product.OdooCode,
		&product.Brend,
		&product.Gs1EAN13,
		&product.Modeli,
		&product.ModelNomi,
	)
	if err == nil {
		return &product, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}
