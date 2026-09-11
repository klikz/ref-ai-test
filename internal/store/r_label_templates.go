package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/klikz/api_v3/internal/models"
)

// Label templates are read on every print — several times per label, since each
// line handler resolves the template again before delegating. They change only
// when an operator edits one, so they are cached and invalidated on write.
//
// The TTL is a safety net for edits made outside this process (e.g. straight in
// the database); LABEL_TEMPLATE_CACHE_TTL_MS overrides it, 0 disables caching.
type cachedLabelTemplate struct {
	template models.LabelTemplate
	expires  time.Time
}

var (
	labelTemplateCacheMu  sync.RWMutex
	labelTemplateCache    = make(map[int]cachedLabelTemplate)
	labelTemplateTTLOnce  sync.Once
	labelTemplateCacheTTL time.Duration
)

func labelTemplateTTL() time.Duration {
	labelTemplateTTLOnce.Do(func() {
		labelTemplateCacheTTL = 15 * time.Second
		if v := os.Getenv("LABEL_TEMPLATE_CACHE_TTL_MS"); v != "" {
			if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
				labelTemplateCacheTTL = time.Duration(ms) * time.Millisecond
			}
		}
	})
	return labelTemplateCacheTTL
}

// cloneLabelTemplate detaches Definition so a caller cannot mutate the cached
// copy through the shared backing array.
func cloneLabelTemplate(t models.LabelTemplate) models.LabelTemplate {
	if len(t.Definition) > 0 {
		definition := make(json.RawMessage, len(t.Definition))
		copy(definition, t.Definition)
		t.Definition = definition
	}
	return t
}

func labelTemplateFromCache(id int) (models.LabelTemplate, bool) {
	if labelTemplateTTL() <= 0 {
		return models.LabelTemplate{}, false
	}
	labelTemplateCacheMu.RLock()
	entry, ok := labelTemplateCache[id]
	labelTemplateCacheMu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return models.LabelTemplate{}, false
	}
	return cloneLabelTemplate(entry.template), true
}

func storeLabelTemplateInCache(id int, template models.LabelTemplate) {
	ttl := labelTemplateTTL()
	if ttl <= 0 {
		return
	}
	labelTemplateCacheMu.Lock()
	labelTemplateCache[id] = cachedLabelTemplate{
		template: cloneLabelTemplate(template),
		expires:  time.Now().Add(ttl),
	}
	labelTemplateCacheMu.Unlock()
}

// invalidateLabelTemplateCache drops one entry, or all of them when id <= 0.
func invalidateLabelTemplateCache(id int) {
	labelTemplateCacheMu.Lock()
	if id > 0 {
		delete(labelTemplateCache, id)
	} else {
		labelTemplateCache = make(map[int]cachedLabelTemplate)
	}
	labelTemplateCacheMu.Unlock()
}

const labelTemplateSelectCols = `
		t.id, t.name, t.line_id, COALESCE(ll.name, ''), t.width_mm, t.height_mm,
		t.dpi, COALESCE(t.print_rotation_deg, 0),
		COALESCE(t.density, 8), COALESCE(t.speed, 4), COALESCE(t.gap_mm, 2),
		COALESCE(t.use_printer_defaults, true), COALESCE(t.size_only, false), t.definition`

func scanLabelTemplate(scanner interface {
	Scan(dest ...any) error
}) (models.LabelTemplate, error) {
	item := models.LabelTemplate{}
	err := scanner.Scan(
		&item.ID, &item.Name, &item.LineID, &item.LineName,
		&item.WidthMm, &item.HeightMm, &item.DPI, &item.PrintRotationDeg,
		&item.Density, &item.Speed, &item.GapMm, &item.UsePrinterDefaults,
		&item.SizeOnly, &item.Definition,
	)
	return item, err
}

func normalizeLabelTemplatePrintSettings(density, speed int, gapMm float64, useDefaults bool) (int, int, float64, bool) {
	// Density: ZPL ~SD 0–30; TSPL DENSITY 0–15 (clamped again at raw encode).
	if density < 0 {
		density = 0
	}
	if density > 30 {
		density = 30
	}
	if speed < 1 {
		speed = 1
	}
	if speed > 14 {
		speed = 14
	}
	if gapMm < 0 {
		gapMm = 0
	}
	if gapMm > 50 {
		gapMm = 50
	}
	return density, speed, gapMm, useDefaults
}

func (r *Repo) LabelTemplatesGetAll() ([]models.LabelTemplate, error) {
	rows, err := r.store.db.Query(`
		SELECT ` + labelTemplateSelectCols + `
		FROM lines.label_templates t
		LEFT JOIN lines.lines_list ll ON ll.line_id = t.line_id
		ORDER BY t.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []models.LabelTemplate{}
	for rows.Next() {
		item, err := scanLabelTemplate(rows)
		if err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	return allData, rows.Err()
}

func (r *Repo) LabelTemplateGetByID(id int) (models.LabelTemplate, error) {
	if cached, ok := labelTemplateFromCache(id); ok {
		return cached, nil
	}

	item, err := scanLabelTemplate(r.store.db.QueryRow(`
		SELECT `+labelTemplateSelectCols+`
		FROM lines.label_templates t
		LEFT JOIN lines.lines_list ll ON ll.line_id = t.line_id
		WHERE t.id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return item, errors.New("label template not found")
		}
		return item, err
	}

	storeLabelTemplateInCache(id, item)
	return cloneLabelTemplate(item), nil
}

func (r *Repo) LabelTemplateCreate(
	name string,
	lineID int,
	widthMm, heightMm float64,
	dpi, printRotationDeg, density, speed int,
	gapMm float64,
	usePrinterDefaults, sizeOnly bool,
	definition json.RawMessage,
	userID int,
) (int, error) {
	if len(definition) == 0 {
		definition = json.RawMessage(`{"version":1,"elements":[]}`)
	}
	density, speed, gapMm, usePrinterDefaults = normalizeLabelTemplatePrintSettings(density, speed, gapMm, usePrinterDefaults)

	var id int
	err := r.store.db.QueryRow(`
		INSERT INTO lines.label_templates
			(name, line_id, width_mm, height_mm, dpi, print_rotation_deg,
			 density, speed, gap_mm, use_printer_defaults, size_only, definition, c_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`,
		name, lineID, widthMm, heightMm, dpi, printRotationDeg,
		density, speed, gapMm, usePrinterDefaults, sizeOnly, definition, userID,
	).Scan(&id)
	if err == nil {
		invalidateLabelTemplateCache(id)
	}
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
		source.Density,
		source.Speed,
		source.GapMm,
		source.UsePrinterDefaults,
		source.SizeOnly,
		definition,
		userID,
	)
}

func (r *Repo) LabelTemplateUpdate(
	id int,
	name string,
	lineID int,
	widthMm, heightMm float64,
	dpi, printRotationDeg, density, speed int,
	gapMm float64,
	usePrinterDefaults, sizeOnly bool,
	definition json.RawMessage,
) error {
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
	density, speed, gapMm, usePrinterDefaults = normalizeLabelTemplatePrintSettings(density, speed, gapMm, usePrinterDefaults)

	res, err := r.store.db.Exec(`
		UPDATE lines.label_templates
		SET name = $1, line_id = $2, width_mm = $3, height_mm = $4, dpi = $5,
		    print_rotation_deg = $6, density = $7, speed = $8, gap_mm = $9,
		    use_printer_defaults = $10, size_only = $11, definition = $12, u_time = NOW()
		WHERE id = $13`,
		name, lineID, widthMm, heightMm, dpi, printRotationDeg,
		density, speed, gapMm, usePrinterDefaults, sizeOnly, definition, id,
	)
	if err != nil {
		return err
	}

	invalidateLabelTemplateCache(id)

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
	invalidateLabelTemplateCache(id)

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("label template not found")
	}
	return nil
}
