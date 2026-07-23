package store

import (
	"database/sql"
	"errors"
	"strings"
)

type LineResponsible struct {
	ID        int64  `json:"id"`
	LineID    int    `json:"line_id"`
	LineName  string `json:"line_name"`
	UserID    int    `json:"user_id"`
	UserName  string `json:"user_name"`
	UserLogin string `json:"user_login"`
	CTime     string `json:"c_time"`
}

func (r *Repo) LineResponsiblesGetAll() ([]LineResponsible, error) {
	rows, err := r.store.db.Query(`
		SELECT lr.id, lr.line_id, ll."name",
			lr.user_id, COALESCE(NULLIF(u.name, ''), u.login, ''), COALESCE(u.login, ''),
			to_char(lr.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM lines.line_responsibles lr
		INNER JOIN lines.lines_list ll ON ll.line_id = lr.line_id
		INNER JOIN auth.users u ON u.id = lr.user_id
		WHERE ll.status = true AND u.status = true
		ORDER BY ll.name, u.login`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LineResponsible{}
	for rows.Next() {
		item := LineResponsible{}
		if err := rows.Scan(
			&item.ID, &item.LineID, &item.LineName,
			&item.UserID, &item.UserName, &item.UserLogin, &item.CTime,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) LineResponsibleAdd(lineID, userID, cUserID int) error {
	if lineID <= 0 || userID <= 0 {
		return errors.New("liniya yoki foydalanuvchi tanlanmagan")
	}
	if cUserID <= 0 {
		return errors.New("foydalanuvchi aniqlanmadi")
	}

	var lineOK int
	err := r.store.db.QueryRow(`
		SELECT 1 FROM lines.lines_list WHERE line_id = $1 AND status = true`, lineID,
	).Scan(&lineOK)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("liniya topilmadi")
		}
		return err
	}

	var userOK int
	err = r.store.db.QueryRow(`
		SELECT 1 FROM auth.users WHERE id = $1 AND status = true`, userID,
	).Scan(&userOK)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("foydalanuvchi topilmadi")
		}
		return err
	}

	_, err = r.store.db.Exec(`
		INSERT INTO lines.line_responsibles (line_id, user_id, c_user_id)
		VALUES ($1, $2, $3)`,
		lineID, userID, cUserID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "line_responsibles_line_user_un") ||
			strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "повторяющееся значение") {
			return errors.New("bu tayinlash allaqachon mavjud")
		}
		return err
	}
	return nil
}

func (r *Repo) LineResponsibleDelete(id int64) error {
	if id <= 0 {
		return errors.New("yozuv topilmadi")
	}
	res, err := r.store.db.Exec(`DELETE FROM lines.line_responsibles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("yozuv topilmadi")
	}
	return nil
}
