package store

import "fmt"

func (r *Repo) EshikNextDailyCounter(eshikModelPartID int) (int, error) {
	if eshikModelPartID <= 0 {
		return 0, fmt.Errorf("eshik qismi noto'g'ri")
	}

	var counter int
	err := r.store.db.QueryRow(`
		INSERT INTO production.eshik_daily_serial (eshik_model_part_id, print_date, counter)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (eshik_model_part_id, print_date)
		DO UPDATE SET counter = production.eshik_daily_serial.counter + 1
		RETURNING counter`,
		eshikModelPartID,
	).Scan(&counter)
	if err != nil {
		return 0, err
	}
	return counter, nil
}

func (r *Repo) EshikDailyCounterUndo(eshikModelPartID int) error {
	if eshikModelPartID <= 0 {
		return nil
	}
	_, err := r.store.db.Exec(`
		UPDATE production.eshik_daily_serial
		SET counter = GREATEST(counter - 1, 0)
		WHERE eshik_model_part_id = $1 AND print_date = CURRENT_DATE`,
		eshikModelPartID,
	)
	return err
}
