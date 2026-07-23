package store

import "fmt"

func (r *Repo) EshikNextDailyCounter(eshikComponentID int) (int, error) {
	if eshikComponentID <= 0 {
		return 0, fmt.Errorf("eshik komponenti noto'g'ri")
	}

	var counter int
	err := r.store.db.QueryRow(`
		INSERT INTO production.eshik_daily_serial (eshik_component_id, print_date, counter)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (eshik_component_id, print_date)
		DO UPDATE SET counter = production.eshik_daily_serial.counter + 1
		RETURNING counter`,
		eshikComponentID,
	).Scan(&counter)
	if err != nil {
		return 0, err
	}
	return counter, nil
}

func (r *Repo) EshikDailyCounterUndo(eshikComponentID int) error {
	if eshikComponentID <= 0 {
		return nil
	}
	_, err := r.store.db.Exec(`
		UPDATE production.eshik_daily_serial
		SET counter = GREATEST(counter - 1, 0)
		WHERE eshik_component_id = $1 AND print_date = CURRENT_DATE`,
		eshikComponentID,
	)
	return err
}
