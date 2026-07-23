package store

import "fmt"

func (r *Repo) KlapanNextDailyCounter(klapanComponentID int) (int, error) {
	if klapanComponentID <= 0 {
		return 0, fmt.Errorf("klapan komponenti noto'g'ri")
	}

	var counter int
	err := r.store.db.QueryRow(`
		INSERT INTO production.klapan_daily_serial (klapan_component_id, print_date, counter)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (klapan_component_id, print_date)
		DO UPDATE SET counter = production.klapan_daily_serial.counter + 1
		RETURNING counter`,
		klapanComponentID,
	).Scan(&counter)
	if err != nil {
		return 0, err
	}
	return counter, nil
}
