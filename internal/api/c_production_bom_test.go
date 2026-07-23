package api

import "testing"

func TestProductionBOMColumnOffsetWithLeadingID(t *testing.T) {
	rows := [][]string{
		exportLikeComponentHeaderRow(),
	}
	offset := productionBOMColumnOffset(rows)
	if offset != 1 {
		t.Fatalf("offset = %d, want 1 for UI export with leading ID column", offset)
	}
}

func TestProductionBOMColumnOffsetTemplateWithoutID(t *testing.T) {
	rows := [][]string{
		productionBOMHeaderRow1(),
	}
	offset := productionBOMColumnOffset(rows)
	if offset != 0 {
		t.Fatalf("offset = %d, want 0 for template without ID column", offset)
	}
}

func TestIsProductionBOMSubHeaderRowWithLeadingID(t *testing.T) {
	subheader := []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "",
		"dona", "Set", "dona", "Set",
	}
	if !isProductionBOMSubHeaderRow(subheader, 1) {
		t.Fatal("expected export sub-header row (dona/Set) to be skipped with bomOffset=1")
	}
}

func TestProductionComponentsFromHeaderRowsSkipsTwoHeaderRows(t *testing.T) {
	typeMap := map[string]int{"import": 1, "local": 2, "production": 3}
	unitMap := map[string]int{"pcs": 10, "dona": 10}
	rows := [][]string{
		{
			"ID",
			"Detal Turi kodi",
			"ODOO kod",
			"Mahsulotning korxona kodi",
			"Manufacturer code",
			"Mahsulotning to'liq nomi (O'zb)",
			"Mahsulotning standart nomi (O'zb)",
			"Mahsulotning standart nomi (Ru)",
			"O`lchov birligi",
			"Xarid turi (import, local yoki production)",
			"Xususiyatlari (O'zb)",
			"photo",
			"comment",
			"Net weight [kg]",
			"",
			"Technological waste",
			"",
		},
		{
			"", "", "", "", "", "", "", "", "", "", "", "", "",
			"dona", "Set", "dona", "Set",
		},
		{},
		{
			"1", "DET", "ODOO-1", "FACT-1", "MFG-1", "Full", "Std", "StdRU",
			"pcs", "import", "spec", "", "note", "1.5", "2", "0.1", "0.2",
		},
		{},
	}

	parsed, errs, ok := productionComponentsFromHeaderRows(rows, typeMap, unitMap)
	if !ok {
		t.Fatal("expected header-based parse")
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(parsed) != 1 {
		t.Fatalf("parsed rows = %d, want 1 (2 headers + empty rows skipped)", len(parsed))
	}
	if parsed[0].item.FactoryCode != "FACT-1" {
		t.Fatalf("factory = %q", parsed[0].item.FactoryCode)
	}
	if parsed[0].row != 4 {
		t.Fatalf("data row number = %d, want 4", parsed[0].row)
	}
}

func exportLikeComponentHeaderRow() []string {
	return []string{
		"ID",
		"Detal Turi kodi",
		"ODOO kod",
		"Mahsulotning korxona kodi",
		"Manufacturer code",
		"Mahsulotning to'liq nomi (O'zb)",
		"Mahsulotning standart nomi (O'zb)",
		"Mahsulotning standart nomi (Ru)",
		"O`lchov birligi",
		"Xarid turi (import, local yoki production)",
		"Xususiyatlari (O'zb)",
	}
}

func TestProductionComponentFromBOMRowWithLeadingID(t *testing.T) {
	typeMap := map[string]int{"import": 1, "local": 2, "production": 3}
	unitMap := map[string]int{"pcs": 10, "dona": 10}
	row := []string{
		"42",
		"DET-001",
		"ODOO-1",
		"FACT-1",
		"MFG-1",
		"Full uz",
		"Std uz",
		"Std ru",
		"pcs",
		"import",
		"Spec uz text",
	}
	item, err := productionComponentFromBOMRow(row, 1, typeMap, unitMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != 42 {
		t.Fatalf("id = %d, want 42", item.ID)
	}
	if item.TypeId != 1 {
		t.Fatalf("type id = %d, want 1", item.TypeId)
	}
	if item.SpecificationUz != "Spec uz text" {
		t.Fatalf("spec = %q", item.SpecificationUz)
	}
}
