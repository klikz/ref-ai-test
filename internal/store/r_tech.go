package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/klikz/api_v3/internal/models"
	"github.com/lib/pq"
)

func (r *Repo) ComponentsGetAll() ([]models.TechComponent, error) {
	rows, err := r.store.db.Query(`SELECT c.id, c.detal_turi_kodi, c.factory_code, c.manufacturer_code, c.full_name_uz, c.standard_name_uz,
		c.full_name_ru, c.standard_name_ru, c.specification_uz, c.comment, ct.name AS type, c.type_id, mu.name AS unit, c.unit_id,
		c.net_weight_pcs, c.net_weight_set, c.technological_waste_pcs, c.technological_waste_set,
		c.available, c.odoo_code, c.photo_path
		FROM production.components c
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE c.status = true
		ORDER BY NULLIF(c.factory_code, ''), NULLIF(c.odoo_code, ''), c.manufacturer_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.TechComponent{}

	for rows.Next() {
		comp := models.TechComponent{}
		if err := rows.Scan(&comp.ID, &comp.DetalTuriKodi, &comp.FactoryCode, &comp.ManufacturerCode, &comp.FullNameUz, &comp.StandardNameUz,
			&comp.FullNameRu, &comp.StandardNameRu, &comp.SpecificationUz, &comp.Comment, &comp.Type, &comp.TypeId, &comp.Unit, &comp.UnitId,
			&comp.NetWeightPcs, &comp.NetWeightSet, &comp.TechnologicalWastePcs, &comp.TechnologicalWasteSet, &comp.Available, &comp.OdooCode,
			&comp.PhotoPath); err != nil {
			return allData, err
		}
		fillTechComponentAliases(&comp)
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) ComponentsGetById(id int) (models.TechComponent, error) {
	data := models.TechComponent{}
	err := r.store.db.QueryRow(`SELECT c.id, c.detal_turi_kodi, c.factory_code, c.manufacturer_code, c.full_name_uz, c.standard_name_uz,
		c.full_name_ru, c.standard_name_ru, c.specification_uz, c.comment, ct.name AS type, c.type_id, mu.name AS unit, c.unit_id,
		c.net_weight_pcs, c.net_weight_set, c.technological_waste_pcs, c.technological_waste_set,
		c.available, c.odoo_code, c.photo_path
		FROM production.components c
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE c.status = true AND c.id = $1`, id).Scan(&data.ID, &data.DetalTuriKodi, &data.FactoryCode, &data.ManufacturerCode, &data.FullNameUz, &data.StandardNameUz,
		&data.FullNameRu, &data.StandardNameRu, &data.SpecificationUz, &data.Comment, &data.Type, &data.TypeId,
		&data.Unit, &data.UnitId, &data.NetWeightPcs, &data.NetWeightSet, &data.TechnologicalWastePcs, &data.TechnologicalWasteSet,
		&data.Available, &data.OdooCode, &data.PhotoPath)
	if err != nil {
		return data, err
	}
	fillTechComponentAliases(&data)
	return data, nil
}

func (r *Repo) ComponentsGetByFactoryCode(factory_code string) (models.TechComponent, error) {
	return r.ComponentsGetByPrimaryKey(factory_code)
}

func (r *Repo) ComponentsGetByPrimaryKey(code string) (models.TechComponent, error) {
	code = cleanComponentCode(code)
	data := models.TechComponent{}
	err := r.store.db.QueryRow(`SELECT c.id, c.detal_turi_kodi, c.factory_code, c.manufacturer_code, c.full_name_uz, c.standard_name_uz,
		c.full_name_ru, c.standard_name_ru, c.specification_uz, c.comment, ct.name AS type, c.type_id, mu.name AS unit, c.unit_id,
		c.net_weight_pcs, c.net_weight_set, c.technological_waste_pcs, c.technological_waste_set,
		c.available, c.odoo_code, c.photo_path
		FROM production.components c
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE c.status = true AND (
			(c.factory_code <> '' AND c.factory_code = $1)
			OR (c.odoo_code <> '' AND c.odoo_code = $1)
			OR c.manufacturer_code = $1
		)
		ORDER BY CASE
			WHEN c.factory_code = $1 THEN 0
			WHEN c.odoo_code = $1 THEN 1
			ELSE 2
		END
		LIMIT 1`, code).Scan(&data.ID, &data.DetalTuriKodi, &data.FactoryCode, &data.ManufacturerCode, &data.FullNameUz,
		&data.StandardNameUz, &data.FullNameRu, &data.StandardNameRu, &data.SpecificationUz, &data.Comment, &data.Type,
		&data.TypeId, &data.Unit, &data.UnitId, &data.NetWeightPcs, &data.NetWeightSet, &data.TechnologicalWastePcs, &data.TechnologicalWasteSet,
		&data.Available, &data.OdooCode, &data.PhotoPath)
	if err != nil {
		return data, err
	}
	fillTechComponentAliases(&data)
	return data, nil
}

func (r *Repo) ComponentsFindExistingKeys(factoryCodes, odooCodes, manufacturerCodes []string) ([]models.TechComponent, error) {
	if len(factoryCodes) == 0 && len(odooCodes) == 0 && len(manufacturerCodes) == 0 {
		return nil, nil
	}

	rows, err := r.store.db.Query(`
		SELECT id, factory_code, manufacturer_code, odoo_code
		FROM production.components
		WHERE status = true
		  AND (
			(factory_code <> '' AND factory_code = ANY($1))
			OR (odoo_code <> '' AND odoo_code = ANY($2))
			OR manufacturer_code = ANY($3)
		  )`,
		pq.Array(factoryCodes),
		pq.Array(odooCodes),
		pq.Array(manufacturerCodes),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.TechComponent{}
	for rows.Next() {
		item := models.TechComponent{}
		if err := rows.Scan(&item.ID, &item.FactoryCode, &item.ManufacturerCode, &item.OdooCode); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) ComponentsUpdate(data models.TechComponent, user_id int) error {
	normalizeTechComponent(&data)

	_, err := r.store.db.Exec(`
		update production.components
		set detal_turi_kodi = $1, factory_code = $2, manufacturer_code = $3, full_name_uz = $4, standard_name_uz = $5,
		full_name_ru = $6, standard_name_ru = $7, specification_uz = $8, net_weight_pcs = $9, net_weight_set = $10,
		technological_waste_pcs = $11, technological_waste_set = $12,
		unit_id = $13, type_id = $14, comment = $15, u_user_id = $16, u_time = now(),
		odoo_code = $17
		where id = $18`,
		data.DetalTuriKodi, data.FactoryCode, data.ManufacturerCode, data.FullNameUz, data.StandardNameUz, data.FullNameRu,
		data.StandardNameRu, data.SpecificationUz, data.NetWeightPcs, data.NetWeightSet, data.TechnologicalWastePcs, data.TechnologicalWasteSet,
		data.UnitId, data.TypeId, data.Comment, user_id, data.OdooCode, data.ID)

	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ComponentsUpdatePhoto(componentID int, photoPath string, userID int) error {
	_, err := r.store.db.Exec(`
		update production.components
		set photo_path = $1, u_user_id = $2, u_time = now()
		where id = $3 and status = true`,
		photoPath,
		userID,
		componentID,
	)
	return err
}

func (r *Repo) TechComponentsInsert(data models.TechComponent, user_id int) error {
	normalizeTechComponent(&data)

	_, err := r.store.db.Exec(`
		insert into production.components (detal_turi_kodi, factory_code, manufacturer_code, full_name_uz, standard_name_uz,
		full_name_ru, standard_name_ru, specification_uz, net_weight_pcs, net_weight_set, technological_waste_pcs, technological_waste_set,
		unit_id, type_id, comment, c_user_id, odoo_code)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		data.DetalTuriKodi, data.FactoryCode, data.ManufacturerCode, data.FullNameUz, data.StandardNameUz, data.FullNameRu,
		data.StandardNameRu, data.SpecificationUz, data.NetWeightPcs, data.NetWeightSet, data.TechnologicalWastePcs, data.TechnologicalWasteSet,
		data.UnitId, data.TypeId, data.Comment, user_id, data.OdooCode)

	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) TechComponentsDelete(id int, user_id int) error {
	_, err := r.store.db.Exec(`
		update production.components 
		set status = false,
			factory_code = CASE
				WHEN factory_code <> '' THEN factory_code || '_' || gen_random_uuid()
				ELSE factory_code
			END,
			manufacturer_code = manufacturer_code || '_' || gen_random_uuid(),
			odoo_code = CASE
				WHEN odoo_code <> '' THEN odoo_code || '_' || gen_random_uuid()
				ELSE odoo_code
			END,
			u_user_id = $2,
			u_time = now()
		where id = $1`, id, user_id)
	if err != nil {
		return err
	}
	return nil
}

type IdName struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (r *Repo) TypesGet() ([]IdName, error) {
	rows, err := r.store.db.Query(`select id, name from production.component_types order by name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []IdName{}

	for rows.Next() {
		comp := IdName{}
		if err := rows.Scan(&comp.Id, &comp.Name); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) UnitsGet() ([]IdName, error) {
	rows, err := r.store.db.Query(`select id, name from production.measurement_units order by name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []IdName{}

	for rows.Next() {
		comp := IdName{}
		if err := rows.Scan(&comp.Id, &comp.Name); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func normalizeTechComponent(data *models.TechComponent) {
	data.DetalTuriKodi = strings.TrimSpace(data.DetalTuriKodi)
	data.FactoryCode = cleanComponentCode(data.FactoryCode)
	data.ManufacturerCode = cleanComponentCode(data.ManufacturerCode)
	data.OdooCode = cleanComponentCode(data.OdooCode)
	if data.ManufacturerCode == "" {
		switch {
		case data.FactoryCode != "":
			data.ManufacturerCode = data.FactoryCode
		default:
			data.ManufacturerCode = cleanComponentCode(data.MainCode)
		}
	}
	if data.FullNameUz == "" {
		data.FullNameUz = data.NameLong
	}
	if data.StandardNameUz == "" {
		data.StandardNameUz = data.NameShortUz
	}
	if data.SpecificationUz == "" {
		data.SpecificationUz = data.SpecsUz
	}
	if data.NetWeightPcs == 0 && data.NetWeightKg != 0 {
		data.NetWeightPcs = data.NetWeightKg
	}
	if data.NetWeightPcs == 0 && data.Weight != 0 {
		data.NetWeightPcs = data.Weight
	}
	if data.TechnologicalWastePcs == 0 && data.TechnologicalWaste != 0 {
		data.TechnologicalWastePcs = data.TechnologicalWaste
	}
	if data.TechnologicalWastePcs == 0 && data.TechWaste != 0 {
		data.TechnologicalWastePcs = data.TechWaste
	}
	if data.StandardNameRu == "" {
		data.StandardNameRu = data.NameRus
	}
	fillTechComponentAliases(data)
}

func cleanComponentCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return ""
	}
	return value
}

func fillTechComponentAliases(data *models.TechComponent) {
	switch {
	case data.FactoryCode != "":
		data.MainCode = data.FactoryCode
	case data.OdooCode != "":
		data.MainCode = data.OdooCode
	default:
		data.MainCode = data.ManufacturerCode
	}
	data.NameLong = data.FullNameUz
	data.NameShortUz = data.StandardNameUz
	data.SpecsUz = data.SpecificationUz
	data.NetWeightKg = data.NetWeightPcs
	data.TechnologicalWaste = data.TechnologicalWastePcs
	data.TechWaste = data.TechnologicalWastePcs
	data.NameRus = data.StandardNameRu
	data.Weight = data.NetWeightPcs
}

func (r *Repo) ModelsAdd(model models.ModelInfo, user_id int) error {
	ApplyModelImportSerialDefaults(&model)
	if err := validateModelAccSerial(model.Acc_serial); err != nil {
		return err
	}
	if err := validateModelCompressorSerial(model.Compressor_serial); err != nil {
		return err
	}

	_, err := r.store.db.Exec(`INSERT INTO production.models (
		seriya_raqami, acc_serial, modeli, sovutgich_turi, qisqa_nomi, rangi, sotuv_turi,
		gs1_ean13, gost, taminot_kuchlanishi_v, xladagent_miqdori_g, energiya_samaradorlik_sarfi,
		kompressor_nomi, maxalliy_sertifikat, eac_sertifikati, ce_sertifikat,
		ishlab_chiqaruvchi_mamlakat, korxon_nomi, manzil, brend, local_export,
		netto, brutto, qadoq_hajmi, mahsulot_hajmi, iqlim_sharoitlari,
		elektr_toki_kuchlanishi_va_turi, yoritgich_lampaning_quvvati_vt, umumiy_hajmi_l,
		sovutgich_kamera_hajmi_l, muzlatgich_kamera_hajmi_l, muzlatish_quvvati,
		nominal_tok_quvvati_w, nominal_tok_kuchi_a, freon, shovqin_darajasi_db, odoo_code, door_code,
		freeze_door_code, ref_door_code,
		compressor_serial, comment,
		eshik_rangi, rangi_eng, korpus_rangi_shortname, eshik_rangi_shortname, rangi_kodi, manzil_ru,
		c_user_id
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,
		$40,$41,$42,$43,$44,$45,$46,$47,$48,$49
	)`,
		model.Seriya_raqami, model.Acc_serial, model.Modeli, model.Sovutgich_turi, model.Qisqa_nomi, model.Rangi, model.Sotuv_turi,
		model.GS1_EAN13, model.GOST, model.Taminot_kuchlanishi_v, model.Xladagent_miqdori_g, model.Energiya_samaradorlik_sarfi,
		model.Kompressor_nomi, model.Maxalliy_sertifikat, model.EAC_Sertifikati, model.CE_Sertifikat,
		model.Ishlab_chiqaruvchi_mamlakat, model.Korxon_nomi, model.Manzil, model.Brend, model.Local_export,
		model.Netto, model.Brutto, model.Qadoq_hajmi, model.Mahsulot_hajmi, model.Iqlim_sharoitlari,
		model.Elektr_toki_kuchlanishi_va_turi, model.Yoritgich_lampaning_quvvati_vt, model.Umumiy_hajmi_l,
		model.Sovutgich_kamera_hajmi_l, model.Muzlatgich_kamera_hajmi_l, model.Muzlatish_quvvati,
		model.Nominal_tok_quvvati_w, model.Nominal_tok_kuchi_a, model.Freon, model.Shovqin_darajasi_db, model.OdooCode, model.Freeze_door_code,
		model.Freeze_door_code, model.Ref_door_code,
		model.Compressor_serial, model.Comment,
		model.EshikRangi, model.RangiEng, model.KorpusRangiShortname, model.EshikRangiShortname, model.RangiKodi, model.ManzilRu,
		user_id)

	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ModelsUpsert(model models.ModelInfo, userID int) (string, error) {
	id, err := r.ModelsFindUpsertID(model)
	if err != nil {
		return "", err
	}
	if id > 0 {
		model.ID = id
		if err := r.ModelsUpdate(model, userID); err != nil {
			return "", err
		}
		return "updated", nil
	}
	if err := r.ModelsAdd(model, userID); err != nil {
		return "", err
	}
	return "inserted", nil
}

func (r *Repo) modelsFindIDByLookupKeys(keys []struct {
	column string
	value  string
}) (int, error) {
	conditions := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		if key.value == "" {
			continue
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", key.column, len(args)+1))
		args = append(args, key.value)
	}
	if len(conditions) == 0 {
		return 0, nil
	}
	query := fmt.Sprintf(
		`SELECT id FROM production.models WHERE deleted = false AND (%s) ORDER BY id LIMIT 1`,
		strings.Join(conditions, " OR "),
	)
	var id int
	err := r.store.db.QueryRow(query, args...).Scan(&id)
	if err == nil {
		return id, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return 0, err
}

func (r *Repo) modelLookupKeys(model models.ModelInfo) []struct {
	column string
	value  string
} {
	return []struct {
		column string
		value  string
	}{
		{column: "odoo_code", value: strings.TrimSpace(model.OdooCode)},
		{column: "seriya_raqami", value: strings.TrimSpace(model.Seriya_raqami)},
		{column: "gs1_ean13", value: strings.TrimSpace(model.GS1_EAN13)},
	}
}

func (r *Repo) ModelsFindUpsertID(model models.ModelInfo) (int, error) {
	if model.ID > 0 {
		var id int
		err := r.store.db.QueryRow(
			`SELECT id FROM production.models WHERE id = $1 AND deleted = false`,
			model.ID,
		).Scan(&id)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
	}

	return r.modelsFindIDByLookupKeys(r.modelLookupKeys(model))
}

func (r *Repo) ModelsFindDuplicateKey(model models.ModelInfo) (int, string, string, error) {
	id, err := r.modelsFindIDByLookupKeys(r.modelLookupKeys(model))
	if err != nil || id == 0 {
		return 0, "", "", err
	}
	var odooCode, seriya, ean13 string
	err = r.store.db.QueryRow(`
		SELECT COALESCE(odoo_code, ''), COALESCE(seriya_raqami, ''),
			COALESCE(gs1_ean13, '')
		FROM production.models
		WHERE id = $1 AND deleted = false`, id,
	).Scan(&odooCode, &seriya, &ean13)
	if err != nil {
		return 0, "", "", err
	}
	stored := map[string]string{
		"odoo_code":     odooCode,
		"seriya_raqami": seriya,
		"gs1_ean13":     ean13,
	}
	for _, key := range r.modelLookupKeys(model) {
		if key.value != "" && stored[key.column] == key.value {
			return id, key.column, key.value, nil
		}
	}
	return id, "", "", nil
}

func (r *Repo) ModelsGetGS1ByID(modelID int) (string, error) {
	var ean string
	err := r.store.db.QueryRow(
		`SELECT COALESCE(gs1_ean13, '') FROM production.models WHERE id = $1 AND deleted = false`,
		modelID,
	).Scan(&ean)
	if err != nil {
		return "", err
	}
	ean = strings.TrimSpace(ean)
	if ean != "" {
		return ean, nil
	}
	return "", errors.New("model uchun GS1 prefiksi topilmadi")
}

func (r *Repo) ModelsGetAll() ([]models.ModelInfo, error) {
	rows, err := r.store.db.Query(`SELECT m.id, m.seriya_raqami, m.acc_serial, m.modeli, m.sovutgich_turi, m.qisqa_nomi, m.rangi, m.sotuv_turi,
	m.gs1_ean13, m.gost, m.taminot_kuchlanishi_v, m.xladagent_miqdori_g, m.energiya_samaradorlik_sarfi,
	m.kompressor_nomi, m.maxalliy_sertifikat, m.eac_sertifikati, m.ce_sertifikat,
	m.ishlab_chiqaruvchi_mamlakat, m.korxon_nomi, m.manzil, m.brend, m.local_export,
	m.netto, m.brutto, m.qadoq_hajmi, m.mahsulot_hajmi, m.iqlim_sharoitlari,
	m.elektr_toki_kuchlanishi_va_turi, m.yoritgich_lampaning_quvvati_vt, m.umumiy_hajmi_l,
	m.sovutgich_kamera_hajmi_l, m.muzlatgich_kamera_hajmi_l, m.muzlatish_quvvati,
	m.nominal_tok_quvvati_w, m.nominal_tok_kuchi_a, m.freon, m.shovqin_darajasi_db, m.odoo_code, m.door_code,
	COALESCE(m.freeze_door_code, ''), COALESCE(m.ref_door_code, ''),
	m.compressor_serial, m.comment,
	m.eshik_rangi, m.rangi_eng, m.korpus_rangi_shortname, m.eshik_rangi_shortname, m.rangi_kodi, m.manzil_ru,
	m.status, COUNT(gs.id) AS gscode_count
	FROM production.models m
	LEFT JOIN production.gscodes gs ON gs.model_id = m.id AND gs.status
	WHERE m.deleted = false
	GROUP BY m.id
	ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.ModelInfo{}
	for rows.Next() {
		comp := models.ModelInfo{}
		if err := rows.Scan(
			&comp.ID, &comp.Seriya_raqami, &comp.Acc_serial, &comp.Modeli, &comp.Sovutgich_turi, &comp.Qisqa_nomi, &comp.Rangi, &comp.Sotuv_turi,
			&comp.GS1_EAN13, &comp.GOST, &comp.Taminot_kuchlanishi_v, &comp.Xladagent_miqdori_g, &comp.Energiya_samaradorlik_sarfi,
			&comp.Kompressor_nomi, &comp.Maxalliy_sertifikat, &comp.EAC_Sertifikati, &comp.CE_Sertifikat,
			&comp.Ishlab_chiqaruvchi_mamlakat, &comp.Korxon_nomi, &comp.Manzil, &comp.Brend, &comp.Local_export,
			&comp.Netto, &comp.Brutto, &comp.Qadoq_hajmi, &comp.Mahsulot_hajmi, &comp.Iqlim_sharoitlari,
			&comp.Elektr_toki_kuchlanishi_va_turi, &comp.Yoritgich_lampaning_quvvati_vt, &comp.Umumiy_hajmi_l,
			&comp.Sovutgich_kamera_hajmi_l, &comp.Muzlatgich_kamera_hajmi_l, &comp.Muzlatish_quvvati,
			&comp.Nominal_tok_quvvati_w, &comp.Nominal_tok_kuchi_a, &comp.Freon, &comp.Shovqin_darajasi_db, &comp.OdooCode, &comp.Door_code,
			&comp.Freeze_door_code, &comp.Ref_door_code,
			&comp.Compressor_serial, &comp.Comment,
			&comp.EshikRangi, &comp.RangiEng, &comp.KorpusRangiShortname, &comp.EshikRangiShortname, &comp.RangiKodi, &comp.ManzilRu,
			&comp.Status, &comp.GsCodeCount,
		); err != nil {
			return allData, err
		}
		if strings.TrimSpace(comp.Freeze_door_code) == "" {
			comp.Freeze_door_code = comp.Door_code
		}
		if strings.TrimSpace(comp.Ref_door_code) == "" {
			comp.Ref_door_code = comp.Door_code
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) ModelsGetByID(modelID int) (models.ModelInfo, error) {
	comp := models.ModelInfo{}
	if modelID <= 0 {
		return comp, nil
	}
	err := r.store.db.QueryRow(`SELECT m.id, m.seriya_raqami, m.acc_serial, m.modeli, m.sovutgich_turi, m.qisqa_nomi, m.rangi, m.sotuv_turi,
	m.gs1_ean13, m.gost, m.taminot_kuchlanishi_v, m.xladagent_miqdori_g, m.energiya_samaradorlik_sarfi,
	m.kompressor_nomi, m.maxalliy_sertifikat, m.eac_sertifikati, m.ce_sertifikat,
	m.ishlab_chiqaruvchi_mamlakat, m.korxon_nomi, m.manzil, m.brend, m.local_export,
	m.netto, m.brutto, m.qadoq_hajmi, m.mahsulot_hajmi, m.iqlim_sharoitlari,
	m.elektr_toki_kuchlanishi_va_turi, m.yoritgich_lampaning_quvvati_vt, m.umumiy_hajmi_l,
	m.sovutgich_kamera_hajmi_l, m.muzlatgich_kamera_hajmi_l, m.muzlatish_quvvati,
	m.nominal_tok_quvvati_w, m.nominal_tok_kuchi_a, m.freon, m.shovqin_darajasi_db, m.odoo_code, m.door_code,
	COALESCE(m.freeze_door_code, ''), COALESCE(m.ref_door_code, ''),
	m.compressor_serial, m.comment,
	m.eshik_rangi, m.rangi_eng, m.korpus_rangi_shortname, m.eshik_rangi_shortname, m.rangi_kodi, m.manzil_ru,
	m.status, COUNT(gs.id) AS gscode_count
	FROM production.models m
	LEFT JOIN production.gscodes gs ON gs.model_id = m.id AND gs.status
	WHERE m.deleted = false AND m.id = $1
	GROUP BY m.id`, modelID).Scan(
		&comp.ID, &comp.Seriya_raqami, &comp.Acc_serial, &comp.Modeli, &comp.Sovutgich_turi, &comp.Qisqa_nomi, &comp.Rangi, &comp.Sotuv_turi,
		&comp.GS1_EAN13, &comp.GOST, &comp.Taminot_kuchlanishi_v, &comp.Xladagent_miqdori_g, &comp.Energiya_samaradorlik_sarfi,
		&comp.Kompressor_nomi, &comp.Maxalliy_sertifikat, &comp.EAC_Sertifikati, &comp.CE_Sertifikat,
		&comp.Ishlab_chiqaruvchi_mamlakat, &comp.Korxon_nomi, &comp.Manzil, &comp.Brend, &comp.Local_export,
		&comp.Netto, &comp.Brutto, &comp.Qadoq_hajmi, &comp.Mahsulot_hajmi, &comp.Iqlim_sharoitlari,
		&comp.Elektr_toki_kuchlanishi_va_turi, &comp.Yoritgich_lampaning_quvvati_vt, &comp.Umumiy_hajmi_l,
		&comp.Sovutgich_kamera_hajmi_l, &comp.Muzlatgich_kamera_hajmi_l, &comp.Muzlatish_quvvati,
		&comp.Nominal_tok_quvvati_w, &comp.Nominal_tok_kuchi_a, &comp.Freon, &comp.Shovqin_darajasi_db, &comp.OdooCode, &comp.Door_code,
		&comp.Freeze_door_code, &comp.Ref_door_code,
		&comp.Compressor_serial, &comp.Comment,
		&comp.EshikRangi, &comp.RangiEng, &comp.KorpusRangiShortname, &comp.EshikRangiShortname, &comp.RangiKodi, &comp.ManzilRu,
		&comp.Status, &comp.GsCodeCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return comp, nil
	}
	if err != nil {
		return comp, err
	}
	if strings.TrimSpace(comp.Freeze_door_code) == "" {
		comp.Freeze_door_code = comp.Door_code
	}
	if strings.TrimSpace(comp.Ref_door_code) == "" {
		comp.Ref_door_code = comp.Door_code
	}
	return comp, nil
}

func (r *Repo) ModelsUpdate(model models.ModelInfo, user_id int) error {
	ApplyModelImportSerialDefaults(&model)
	if err := validateModelAccSerial(model.Acc_serial); err != nil {
		return err
	}
	if err := validateModelCompressorSerial(model.Compressor_serial); err != nil {
		return err
	}

	_, err := r.store.db.Exec(`
	UPDATE production.models SET
		seriya_raqami=$1, acc_serial=$2, modeli=$3, sovutgich_turi=$4, qisqa_nomi=$5, rangi=$6, sotuv_turi=$7,
		gs1_ean13=$8, gost=$9, taminot_kuchlanishi_v=$10, xladagent_miqdori_g=$11, energiya_samaradorlik_sarfi=$12,
		kompressor_nomi=$13, maxalliy_sertifikat=$14, eac_sertifikati=$15, ce_sertifikat=$16,
		ishlab_chiqaruvchi_mamlakat=$17, korxon_nomi=$18, manzil=$19, brend=$20, local_export=$21,
		netto=$22, brutto=$23, qadoq_hajmi=$24, mahsulot_hajmi=$25, iqlim_sharoitlari=$26,
		elektr_toki_kuchlanishi_va_turi=$27, yoritgich_lampaning_quvvati_vt=$28, umumiy_hajmi_l=$29,
		sovutgich_kamera_hajmi_l=$30, muzlatgich_kamera_hajmi_l=$31, muzlatish_quvvati=$32,
		nominal_tok_quvvati_w=$33, nominal_tok_kuchi_a=$34, freon=$35, shovqin_darajasi_db=$36, odoo_code=$37, door_code=$38,
		freeze_door_code=$39, ref_door_code=$40,
		compressor_serial=$41, comment=$42,
		eshik_rangi=$43, rangi_eng=$44, korpus_rangi_shortname=$45, eshik_rangi_shortname=$46, rangi_kodi=$47, manzil_ru=$48,
		u_user_id=$49, u_time=now()
	WHERE id=$50`,
		model.Seriya_raqami, model.Acc_serial, model.Modeli, model.Sovutgich_turi, model.Qisqa_nomi, model.Rangi, model.Sotuv_turi,
		model.GS1_EAN13, model.GOST, model.Taminot_kuchlanishi_v, model.Xladagent_miqdori_g, model.Energiya_samaradorlik_sarfi,
		model.Kompressor_nomi, model.Maxalliy_sertifikat, model.EAC_Sertifikati, model.CE_Sertifikat,
		model.Ishlab_chiqaruvchi_mamlakat, model.Korxon_nomi, model.Manzil, model.Brend, model.Local_export,
		model.Netto, model.Brutto, model.Qadoq_hajmi, model.Mahsulot_hajmi, model.Iqlim_sharoitlari,
		model.Elektr_toki_kuchlanishi_va_turi, model.Yoritgich_lampaning_quvvati_vt, model.Umumiy_hajmi_l,
		model.Sovutgich_kamera_hajmi_l, model.Muzlatgich_kamera_hajmi_l, model.Muzlatish_quvvati,
		model.Nominal_tok_quvvati_w, model.Nominal_tok_kuchi_a, model.Freon, model.Shovqin_darajasi_db, model.OdooCode, model.Freeze_door_code,
		model.Freeze_door_code, model.Ref_door_code,
		model.Compressor_serial, model.Comment,
		model.EshikRangi, model.RangiEng, model.KorpusRangiShortname, model.EshikRangiShortname, model.RangiKodi, model.ManzilRu,
		user_id, model.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ModelsUpdateWithStatus(model models.ModelInfo, user_id int) error {
	if err := r.ModelsUpdate(model, user_id); err != nil {
		return err
	}
	_, err := r.store.db.Exec(
		`UPDATE production.models SET status=$1, u_user_id=$2, u_time=now() WHERE id=$3`,
		model.Status,
		user_id,
		model.ID,
	)
	return err
}

func (r *Repo) ModelsChangeStatus(model_id int, user_id int) error {
	_, err := r.store.db.Exec(`UPDATE production.models SET status= NOT status, u_user_id=$1, u_time=now() WHERE id=$2`, user_id, model_id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ModelsUpdateCount(model_id int) (int, error) {

	count := 0
	err := r.store.db.QueryRow(`
	UPDATE production.models t
		SET serial_count = case when t.last_serial_time =now()::date then t.serial_count +1 else 1 end,
			last_serial_time =  case when t.last_serial_time =now()::date then t.last_serial_time else now() end
		WHERE t.id  = $1
	returning t.serial_count
	`, model_id).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repo) ModelsUpdateCountMinus(model_id int) (int, error) {

	count := 0
	err := r.store.db.QueryRow(`
	UPDATE production.models t
		SET serial_count = case when t.last_serial_time =now()::date then t.serial_count - 1 else 1 end,
			last_serial_time =  case when t.last_serial_time =now()::date then t.last_serial_time else now() end
		WHERE t.id  = $1
	returning t.serial_count
	`, model_id).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repo) GsCodeAdd(gscode string, model_id, product_id, user_id int) error {
	if product_id == 0 {
		_, err := r.store.db.Exec(`INSERT INTO production.gscodes (model_id, "data", c_user_id) VALUES ($1, $2, $3)`, model_id, gscode, user_id)
		if err != nil {
			if strings.Contains(err.Error(), "gscodes_un") {
				return errors.New("gs_code kiritilgan")
			}
			return err
		}
	} else {
		_, err := r.store.db.Exec(`INSERT INTO production.gscodes (model_id, "data", product_id, c_user_id, u_user_id, status, u_time) VALUES ($1, $2, $3, $4, $5, false, now())`, model_id, gscode, product_id, user_id, user_id)
		if err != nil {
			fmt.Println(err.Error())
			if strings.Contains(err.Error(), "gscodes_un") {
				return errors.New("gs_code kiritilgan")
			}
			return err
		}
	}

	return nil
}

func (r *Repo) GsCodeUpdate(model_id, product_id, user_id int) (int, error) {

	id := 0
	err := r.store.db.QueryRow(`WITH cte AS (
			SELECT id
			FROM production.gscodes
			WHERE model_id = $3
			AND status = true
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE production.gscodes AS g
		SET u_user_id = $1,
			u_time = now(),
			product_id = $2,
			status = false
		FROM cte
		WHERE g.id = cte.id
		RETURNING g.id;
		`, user_id, product_id, model_id).Scan(&id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return 0, errors.New("gs_code topilmadi")
		}
		return 0, err
	}

	return id, nil
}

func (r *Repo) GsCodeUpdateUndo(id int) error {

	_, err := r.store.db.Exec(`update production.gscodes g 
		set status = true,
		u_time = null,
		u_user_id = null,
		product_id = null
		where g.id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) LinesGetInfoByProductId(product_id int) (models.ModelInfo, error) {
	info := models.ModelInfo{}

	err := r.store.db.QueryRow(`select p.serial , m.qisqa_nomi, m.modeli, m.odoo_code, m.brend, g."data", 
	to_char(g.u_time, 'YYYY-MM-DD H24:MI') as time
	from production.gscodes g, lines.products p, production.models m 
	where p.id = g.product_id 
	and m.id = g.model_id 
	and g.product_id = $1`, product_id).Scan(&info.SerialNumber, &info.Qisqa_nomi, &info.Modeli,
		&info.OdooCode, &info.Brend, &info.GS1Data, &info.UTime)
	if err != nil {
		return info, err
	}

	return info, nil
}

func (r *Repo) GsCodeBySerial(serial string) (string, error) {
	var data string
	err := r.store.db.QueryRow(`
		SELECT COALESCE(g.data, '')
		FROM lines.products p
		LEFT JOIN production.gscodes g ON g.product_id = p.id
		WHERE p.serial = $1
		ORDER BY g.u_time DESC NULLS LAST
		LIMIT 1`, serial).Scan(&data)
	if err != nil {
		return "", err
	}
	return data, nil
}

type GsCode struct {
	ModelId       int    `json:"model_id"`
	Brand         string `json:"brand"`
	Seriya_raqami string `json:"seriya_raqami"`
	Modeli        string `json:"modeli"`
	ModelNomi     string `json:"model_nomi"`
	Count         int    `json:"count"`
	GS1EAN        string `json:"gs1_ean13"`
}

func (r *Repo) GsCodesGetCount() ([]GsCode, error) {
	rows, err := r.store.db.Query(`select m.id, m.brend, m.seriya_raqami , count(g.id) as quantity, m.modeli, m.qisqa_nomi, m.gs1_ean13  
	from production.gscodes g , production.models m 
	where g.status = true and m.id = g.model_id  
	group by m.id, m.seriya_raqami, m.brend
	order by m.brend, m.seriya_raqami`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []GsCode{}

	for rows.Next() {
		comp := GsCode{}
		if err := rows.Scan(&comp.ModelId, &comp.Brand, &comp.Seriya_raqami, &comp.Count, &comp.Modeli, &comp.ModelNomi, &comp.GS1EAN); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

type GsCodeReportStruct struct {
	YuklanganSana int    `json:"yuklangan"`
	Ishatilgan    int    `json:"ishlatilgan"`
	Qoldiq        int    `json:"qoldiq"`
	ModelId       int    `json:"model_id"`
	Modeli        string `json:"modeli"`
	SeriyaRaqami  string `json:"seriya_raqami"`
	ArtikulRaqami string `json:"artikul_raqami"`
	OdooCode      string `json:"odoo_code"`
}

func (r *Repo) GsCodesReport(date1, date2 string) ([]GsCodeReportStruct, error) {
	rows, err := r.store.db.Query(`SELECT COUNT(*) AS yuklangan,    
		COUNT(CASE WHEN g.u_time >= COALESCE(NULLIF($1, '')::date, CURRENT_DATE) 
		AND g.u_time < COALESCE(NULLIF($2, '')::date, CURRENT_DATE) + INTERVAL '1 day' THEN 1 END) AS ishlatilgan,    
		COUNT(CASE WHEN g.status = true THEN 1 END ) AS qoldiq,    
		g.model_id, m.modeli,
		COALESCE(m.seriya_raqami, ''), COALESCE(m.qisqa_nomi, ''), COALESCE(m.odoo_code, '')
		FROM production.gscodes g
		JOIN production.models m     
		ON m.id = g.model_id
		WHERE     
		g.c_time >= COALESCE(NULLIF($1, '')::date, CURRENT_DATE)    
		AND g.c_time < COALESCE(NULLIF($2, '')::date, CURRENT_DATE) + INTERVAL '1 day'
		GROUP BY g.model_id, m.modeli, m.seriya_raqami, m.qisqa_nomi, m.odoo_code
		ORDER BY g.model_id;`, date1, date2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []GsCodeReportStruct{}

	for rows.Next() {
		comp := GsCodeReportStruct{}
		if err := rows.Scan(
			&comp.YuklanganSana,
			&comp.Ishatilgan,
			&comp.Qoldiq,
			&comp.ModelId,
			&comp.Modeli,
			&comp.SeriyaRaqami,
			&comp.ArtikulRaqami,
			&comp.OdooCode,
		); err != nil {
			return allData, err
		}
		allData = append(allData, comp)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

type ModelShortInfo struct {
	ModelId          int    `json:"model_id"`
	ArtikulRaqami    string `json:"artikul_raqami"`
	ModelName        string `json:"model_name"`
	Gs1Shablon       string `json:"gs1_shablon"`
	ImportCode       string `json:"import_code"`
	SeriyaRaqami     string `json:"seriya_raqami"`
	AccSerial        string `json:"acc_serial"`
	CompressorSerial string `json:"compressor_serial"`
	DoorCode         string `json:"door_code"` // legacy alias of freeze
	FreezeDoorCode   string `json:"freeze_door_code"`
	RefDoorCode      string `json:"ref_door_code"`
	QisqaNomi        string `json:"qisqa_nomi"`
	GS1EAN13         string `json:"gs1_ean13"`
}

func (r *Repo) ModelsShortInfoBySerial(serial string) (ModelShortInfo, error) {
	return r.ModelsShortInfoBySerialPrefix(serial, 7)
}

func (r *Repo) ModelsShortInfoBySerialPrefix(serial string, prefixLen int) (ModelShortInfo, error) {
	if prefixLen <= 0 {
		prefixLen = 7
	}
	if len(serial) < prefixLen {
		return ModelShortInfo{}, errors.New("serial noto'g'ri")
	}

	croppedSerial := serial[:prefixLen]
	modelInfo := ModelShortInfo{}

	err := r.store.db.QueryRow(`select m.id, COALESCE(m.qisqa_nomi, ''), m.modeli, COALESCE(m.gs1_ean13, ''),
								COALESCE(NULLIF(m.freeze_door_code, ''), m.door_code, ''),
								COALESCE(NULLIF(m.ref_door_code, ''), m.door_code, ''),
								m.seriya_raqami,
								COALESCE(m.acc_serial, ''), COALESCE(m.compressor_serial, '')
								from production.models m 
								where m.status = true and m.seriya_raqami = $1`, croppedSerial).Scan(
		&modelInfo.ModelId,
		&modelInfo.QisqaNomi, &modelInfo.ModelName, &modelInfo.GS1EAN13,
		&modelInfo.FreezeDoorCode, &modelInfo.RefDoorCode,
		&modelInfo.SeriyaRaqami, &modelInfo.AccSerial,
		&modelInfo.CompressorSerial)
	if err != nil {
		if err == sql.ErrNoRows {
			return ModelShortInfo{}, errors.New("serial ma'lumoti topilmadi")
		}
		return modelInfo, err
	}
	modelInfo.ArtikulRaqami = modelInfo.QisqaNomi
	modelInfo.Gs1Shablon = modelInfo.GS1EAN13
	modelInfo.DoorCode = modelInfo.FreezeDoorCode
	modelInfo.ImportCode = modelInfo.FreezeDoorCode
	return modelInfo, nil
}

func (r *Repo) PrintersAdd(line_id int, name, serial string) (int, error) {
	id := 0
	err := r.store.db.QueryRow(`
		insert into lines.printers (line_id, address, printer_name) values ($1, $2, $3)
		returning id`, line_id, name, serial).Scan(&id)
	if err != nil {
		return id, err
	}
	return id, nil
}

func (r *Repo) PrintersDelete(id int) error {
	_, err := r.store.db.Exec(`delete from lines.printers where id=$1`, id)
	if err != nil {
		return err
	}
	return nil
}
