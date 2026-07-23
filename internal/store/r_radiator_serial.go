package store

import "fmt"

func (r *Repo) RadiatorNextDailyCounter(radiatorComponentID int) (int, error) {
	if radiatorComponentID <= 0 {
		return 0, fmt.Errorf("radiator komponenti noto'g'ri")
	}

	var counter int
	err := r.store.db.QueryRow(`
		INSERT INTO production.radiator_daily_serial (radiator_component_id, print_date, counter)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (radiator_component_id, print_date)
		DO UPDATE SET counter = production.radiator_daily_serial.counter + 1
		RETURNING counter`,
		radiatorComponentID,
	).Scan(&counter)
	if err != nil {
		return 0, err
	}
	return counter, nil
}
