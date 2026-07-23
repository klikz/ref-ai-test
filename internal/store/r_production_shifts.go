package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProductionShiftSettings struct {
	Shift1Start string `json:"shift1_start"`
	Shift1End   string `json:"shift1_end"`
	Shift2Start string `json:"shift2_start"`
	Shift2End   string `json:"shift2_end"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	UpdatedBy   int    `json:"updated_by,omitempty"`
}

type ProductionCurrentShiftInfo struct {
	ShiftNo   int    `json:"shift_no"`
	PlanDate  string `json:"plan_date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Label     string `json:"label"`
}

func defaultProductionShiftSettings() ProductionShiftSettings {
	return ProductionShiftSettings{
		Shift1Start: "08:00",
		Shift1End:   "20:00",
		Shift2Start: "20:00",
		Shift2End:   "08:00",
	}
}

func normalizeShiftTime(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("smena vaqti bo'sh")
	}
	parts := strings.Split(raw, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return "", fmt.Errorf("smena vaqti noto'g'ri: %s", raw)
	}
	hour := 0
	minute := 0
	if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil {
		return "", fmt.Errorf("smena vaqti noto'g'ri: %s", raw)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil {
		return "", fmt.Errorf("smena vaqti noto'g'ri: %s", raw)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "", fmt.Errorf("smena vaqti noto'g'ri: %s", raw)
	}
	return fmt.Sprintf("%02d:%02d", hour, minute), nil
}

func parseShiftClock(raw string) (time.Duration, error) {
	normalized, err := normalizeShiftTime(raw)
	if err != nil {
		return 0, err
	}
	parts := strings.Split(normalized, ":")
	hour := 0
	minute := 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)
	return time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute, nil
}

func productionShiftWindowForDate(planDate string, startRaw, endRaw string) (time.Time, time.Time, error) {
	day, err := productionPlanParseDate(planDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	startOffset, err := parseShiftClock(startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endOffset, err := parseShiftClock(endRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := day.Add(startOffset)
	end := day.Add(endOffset)
	if !end.After(start) {
		end = end.Add(24 * time.Hour)
	}
	return start, end, nil
}

func (s ProductionShiftSettings) ShiftWindow(planDate string, shiftNo int) (time.Time, time.Time, error) {
	switch shiftNo {
	case 1:
		return productionShiftWindowForDate(planDate, s.Shift1Start, s.Shift1End)
	case 2:
		return productionShiftWindowForDate(planDate, s.Shift2Start, s.Shift2End)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("noto'g'ri smena: %d", shiftNo)
	}
}

func (s ProductionShiftSettings) Assign(at time.Time) (shiftNo int, planDate string) {
	loc := at.Location()
	today := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	candidates := []struct {
		shiftNo  int
		planDate string
	}{
		{1, today.Format("2006-01-02")},
		{2, today.Format("2006-01-02")},
		{1, yesterday.Format("2006-01-02")},
		{2, yesterday.Format("2006-01-02")},
	}

	for _, c := range candidates {
		start, end, err := s.ShiftWindow(c.planDate, c.shiftNo)
		if err != nil {
			continue
		}
		if !at.Before(start) && at.Before(end) {
			return c.shiftNo, c.planDate
		}
	}
	return 0, ""
}

func (s ProductionShiftSettings) Current(at time.Time) ProductionCurrentShiftInfo {
	shiftNo, planDate := s.Assign(at)
	info := ProductionCurrentShiftInfo{
		ShiftNo:  shiftNo,
		PlanDate: planDate,
	}
	if shiftNo == 0 || planDate == "" {
		return info
	}
	start, end, err := s.ShiftWindow(planDate, shiftNo)
	if err != nil {
		return info
	}
	info.StartTime = start.Format("15:04")
	info.EndTime = end.Format("15:04")
	info.Label = fmt.Sprintf("%d-sm smena", shiftNo)
	return info
}

func (s ProductionShiftSettings) Validate() error {
	s1s, err := normalizeShiftTime(s.Shift1Start)
	if err != nil {
		return err
	}
	s1e, err := normalizeShiftTime(s.Shift1End)
	if err != nil {
		return err
	}
	s2s, err := normalizeShiftTime(s.Shift2Start)
	if err != nil {
		return err
	}
	s2e, err := normalizeShiftTime(s.Shift2End)
	if err != nil {
		return err
	}
	if s1s == s1e {
		return errors.New("1-smena boshlanish va tugash bir xil bo'lishi mumkin emas")
	}
	if s2s == s2e {
		return errors.New("2-smena boshlanish va tugash bir xil bo'lishi mumkin emas")
	}
	return nil
}

func (r *Repo) ProductionShiftSettingsGet() (ProductionShiftSettings, error) {
	settings := defaultProductionShiftSettings()
	var updatedAt sql.NullTime
	var updatedBy sql.NullInt64
	err := r.store.db.QueryRow(`
		SELECT
			to_char(shift1_start, 'HH24:MI'),
			to_char(shift1_end, 'HH24:MI'),
			to_char(shift2_start, 'HH24:MI'),
			to_char(shift2_end, 'HH24:MI'),
			updated_at,
			updated_by
		FROM production.shift_settings
		WHERE id = 1`).Scan(
		&settings.Shift1Start,
		&settings.Shift1End,
		&settings.Shift2Start,
		&settings.Shift2End,
		&updatedAt,
		&updatedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultProductionShiftSettings(), nil
		}
		return settings, err
	}
	if updatedAt.Valid {
		settings.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
	}
	if updatedBy.Valid {
		settings.UpdatedBy = int(updatedBy.Int64)
	}
	return settings, nil
}

func (r *Repo) ProductionShiftSettingsSave(settings ProductionShiftSettings, userID int) (ProductionShiftSettings, error) {
	s1s, err := normalizeShiftTime(settings.Shift1Start)
	if err != nil {
		return settings, err
	}
	s1e, err := normalizeShiftTime(settings.Shift1End)
	if err != nil {
		return settings, err
	}
	s2s, err := normalizeShiftTime(settings.Shift2Start)
	if err != nil {
		return settings, err
	}
	s2e, err := normalizeShiftTime(settings.Shift2End)
	if err != nil {
		return settings, err
	}
	settings.Shift1Start = s1s
	settings.Shift1End = s1e
	settings.Shift2Start = s2s
	settings.Shift2End = s2e
	if err := settings.Validate(); err != nil {
		return settings, err
	}

	var updatedBy any
	if userID > 0 {
		updatedBy = userID
	} else {
		updatedBy = nil
	}

	_, err = r.store.db.Exec(`
		INSERT INTO production.shift_settings
			(id, shift1_start, shift1_end, shift2_start, shift2_end, updated_at, updated_by)
		VALUES (1, $1::time, $2::time, $3::time, $4::time, NOW(), $5)
		ON CONFLICT (id) DO UPDATE SET
			shift1_start = EXCLUDED.shift1_start,
			shift1_end = EXCLUDED.shift1_end,
			shift2_start = EXCLUDED.shift2_start,
			shift2_end = EXCLUDED.shift2_end,
			updated_at = NOW(),
			updated_by = EXCLUDED.updated_by`,
		s1s, s1e, s2s, s2e, updatedBy,
	)
	if err != nil {
		return settings, err
	}
	return r.ProductionShiftSettingsGet()
}

func (r *Repo) ProductionCurrentShift(at time.Time) (ProductionCurrentShiftInfo, error) {
	settings, err := r.ProductionShiftSettingsGet()
	if err != nil {
		return ProductionCurrentShiftInfo{}, err
	}
	if at.IsZero() {
		at = time.Now()
	}
	return settings.Current(at), nil
}

func (r *Repo) ProductionShiftWindow(planDate string, shiftNo int) (time.Time, time.Time, error) {
	settings, err := r.ProductionShiftSettingsGet()
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return settings.ShiftWindow(planDate, shiftNo)
}
