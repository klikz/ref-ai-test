package models

type ModelInfo struct {
	ID                             int    `json:"id"`
	Seriya_raqami                  string `json:"seriya_raqami"`
	Acc_serial                     string `json:"acc_serial"`
	Modeli                         string `json:"modeli"`
	Sovutgich_turi                 string `json:"sovutgich_turi"`
	Qisqa_nomi                     string `json:"qisqa_nomi"`
	Rangi                          string `json:"rangi"`
	Sotuv_turi                     string `json:"sotuv_turi"`
	GS1_EAN13                      string `json:"gs1_ean13"`
	GOST                           string `json:"gost"`
	Taminot_kuchlanishi_v          string `json:"taminot_kuchlanishi_v"`
	Xladagent_miqdori_g            string `json:"xladagent_miqdori_g"`
	Energiya_samaradorlik_sarfi    string `json:"energiya_samaradorlik_sarfi"`
	Kompressor_nomi                string `json:"kompressor_nomi"`
	Maxalliy_sertifikat            string `json:"maxalliy_sertifikat"`
	EAC_Sertifikati                string `json:"eac_sertifikati"`
	CE_Sertifikat                  string `json:"ce_sertifikat"`
	Ishlab_chiqaruvchi_mamlakat    string `json:"ishlab_chiqaruvchi_mamlakat"`
	Korxon_nomi                    string `json:"korxon_nomi"`
	Manzil                         string `json:"manzil"`
	Brend                          string `json:"brend"`
	Local_export                   string `json:"local_export"`
	Netto                          string `json:"netto"`
	Brutto                         string `json:"brutto"`
	Qadoq_hajmi                    string `json:"qadoq_hajmi"`
	Mahsulot_hajmi                 string `json:"mahsulot_hajmi"`
	Iqlim_sharoitlari              string `json:"iqlim_sharoitlari"`
	Elektr_toki_kuchlanishi_va_turi string `json:"elektr_toki_kuchlanishi_va_turi"`
	Yoritgich_lampaning_quvvati_vt string `json:"yoritgich_lampaning_quvvati_vt"`
	Umumiy_hajmi_l                 string `json:"umumiy_hajmi_l"`
	Sovutgich_kamera_hajmi_l       string `json:"sovutgich_kamera_hajmi_l"`
	Muzlatgich_kamera_hajmi_l      string `json:"muzlatgich_kamera_hajmi_l"`
	Muzlatish_quvvati              string `json:"muzlatish_quvvati"`
	Nominal_tok_quvvati_w          string `json:"nominal_tok_quvvati_w"`
	Freon                          string `json:"freon"`
	Shovqin_darajasi_db            string `json:"shovqin_darajasi_db"`
	OdooCode                       string `json:"odoo_code"`
	Door_code                      string `json:"door_code"`
	Compressor_serial              string `json:"compressor_serial"`
	Comment                        string `json:"comment"`
	Status                         bool   `json:"status"`
	SerialNumber                   string `json:"serial_number"`
	GS1Data                        string `json:"gs1_data"`
	UTime                          string `json:"u_time"`
	GsCodeCount                    int    `json:"gscode_count"`
}

type ModelsCount struct {
	LineID       int    `json:"line_id"`
	LineName     string `json:"line_name"`
	ModelID      int    `json:"model_id"`
	ModelName    string `json:"model_name"`
	SeriyaRaqami string `json:"seriya_raqami"`
	OdooCode     string `json:"odoo_code"`
	Count        int    `json:"count"`
}
