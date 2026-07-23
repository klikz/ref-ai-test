package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/klikz/api_v3/internal/models"
)

func (r *Repo) GetUserById(u *models.User) error {
	if err := r.store.db.QueryRow(`select u.id, u."name", u.login, r."name", u."password", u.role_id 
									from auth.users u, auth.roles r 
									where u.id = $1 and u.status = true and r.id = u.role_id`, u.ID).Scan(
		&u.ID, &u.UserName, &u.Login, &u.Role, &u.EncryptedPassword, &u.Role_ID); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return errors.New("user not found")
		}
		return err
	}
	return nil
}

func (r *Repo) GetUserPermissionsById(id int) (interface{}, error) {
	rows, err := r.store.db.Query(`
		select r.id, 
	   	r.route, 
	   	coalesce (r."comment", 'no comment') as comment,
		case when p.user_id is not null then true else false end as is_flag
		from auth.routes r
		left join auth.permissions p
		on p.route_id = r.id
		and p.user_id = $1
		where r.is_hidden = false
		order by r.route`, id)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var datas []models.Permissions

	for rows.Next() {
		var data models.Permissions
		if err := rows.Scan(
			&data.ID, &data.Route, &data.Comment, &data.IsFlag); err != nil {
			return nil, err
		}
		datas = append(datas, data)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return datas, nil
}

func (r *Repo) Create(u *models.User) error {

	if err := r.store.db.QueryRow(`
		insert into auth.users (login, "name", role_id, "password") values ($1, $2, $3, $4)
		returning "id"`, u.Login, u.UserName, u.Role_ID, u.EncryptedPassword).Scan(&u.ID); err != nil {
		if strings.Contains(err.Error(), `users_unique_1`) {
			return errors.New("bunday ism kiritilgan")
		}
		if strings.Contains(err.Error(), `users_unique`) {
			return errors.New("bunday login kiritilgan")
		}
		return err
	}

	return nil
}

func (r *Repo) GetRoleIDByName(name string) (float64, error) {
	var id float64
	err := r.store.db.QueryRow(`select id from auth.roles where lower("name") = lower($1) limit 1`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err := r.store.db.QueryRow(
		`insert into auth.roles ("name") values ($1) returning id`,
		name,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repo) UpdatePassword(u *models.User) error {
	_, err := r.store.db.Exec(`update auth.users
								set "password" = $1
								where id =  $2`, u.EncryptedPassword, u.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) DeleteUser(user_id int) error {
	_, err := r.store.db.Exec(`update auth.users
		set login = login || '_' || gen_random_uuid(),
		"name" = "name" || '_' || gen_random_uuid(),
		status = false
		where id = $1`, user_id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) UpdateUserInfo(u *models.User) error {
	query := `
	update auth.users 
	set login = $1, "name" = $2`
	args := []any{u.Login, u.UserName}
	if u.Role_ID > 0 {
		query += `, role_id = $3 where id = $4`
		args = append(args, u.Role_ID, u.ID)
	} else {
		query += ` where id = $3`
		args = append(args, u.ID)
	}
	_, err := r.store.db.Exec(query, args...)
	if err != nil {
		if strings.Contains(err.Error(), `users_unique_1`) {
			return errors.New("bunday ism kiritilgan")
		}
		if strings.Contains(err.Error(), `users_unique`) {
			return errors.New("bunday login kiritilgan")
		}
		return err
	}

	return nil
}

func (r *Repo) FindByLogin(u *models.User) error {
	if err := r.store.db.QueryRow(`select u.id, u."name", u.login, r."name", u."password", u.role_id
									from auth.users u, auth.roles r 
									where u.login = $1 and u.status = true and r.id = u.role_id`, u.Login).Scan(
		&u.ID, &u.UserName, &u.Login, &u.Role, &u.EncryptedPassword, &u.Role_ID); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return errors.New("user not found")
		}
		return err
	}
	return nil
}

func (r *Repo) CheckUserRoutePermission(userID int, route string) (bool, error) {
	var ok int
	err := r.store.db.QueryRow(`
		SELECT 1 FROM auth.permissions p
		INNER JOIN auth.routes r ON r.id = p.route_id
		WHERE p.user_id = $1 AND r.route = $2
		LIMIT 1`, userID, route).Scan(&ok)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return false, errors.New("no permission")
		}
		return false, err
	}
	return true, nil
}

func (r *Repo) EnsureRouteAndCheckPermission(userID int, route string) (bool, error) {
	var ok bool
	err := r.store.db.QueryRow(`
		WITH inserted_route AS (
			INSERT INTO auth.routes (route)
			VALUES ($1)
			ON CONFLICT (route) DO NOTHING
			RETURNING id
		),
		selected_route AS (
			SELECT id FROM inserted_route
			UNION ALL
			SELECT r.id
			FROM auth.routes r
			WHERE r.route = $1
			  AND NOT EXISTS (SELECT 1 FROM inserted_route)
		)
		SELECT EXISTS (
			SELECT 1
			FROM auth.permissions p
			INNER JOIN selected_route r ON r.id = p.route_id
			WHERE p.user_id = $2
		)`, route, userID).Scan(&ok)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("route not found")
		}
		return false, err
	}
	if !ok {
		return false, errors.New("no permission")
	}
	return true, nil
}

func (r *Repo) CheckRole(route_id, user_id int) (bool, error) {

	id := 0
	err := r.store.db.QueryRow(`select p.id from auth.permissions p 
								where p.route_id = $1
								and p.user_id = $2`, route_id, user_id).Scan(&id)

	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return false, errors.New("no permission")
		}
		return false, err
	}

	if id == 0 {
		return false, errors.New("no permission")
	}
	return true, nil
}

func (r *Repo) GetUserID(login string) (int, error) {
	id := 0
	if err := r.store.db.QueryRow(`select u.id from auth.users u 
									where u.login = $1 and u.status = true`, login).Scan(&id); err != nil {
		return id, err
	}
	return id, nil
}

func (r *Repo) GetRouteID(route string) (int, error) {
	id := 0
	err := r.store.db.QueryRow(`select r.id from auth.routes r where r.route = $1`, route).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !strings.Contains(err.Error(), "no rows in result set") {
		return 0, err
	}
	if err := r.store.db.QueryRow(
		`insert into auth.routes (route) values ($1) returning id`, route,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repo) SaveToken(token string, user_id int) error {
	_, err := r.store.db.Exec(`update auth.users set "token" = $1 where id= $2`, token, user_id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) GetAllUsers() (interface{}, error) {

	rows, err := r.store.db.Query(`
		select u.id, u."name", u.login, r."name", u.role_id, u.status
		from auth.users u, auth.roles r 
		where r.id = u.role_id
		and u.status = true
		order by u.login `)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var datas []models.User

	for rows.Next() {
		var data models.User
		if err := rows.Scan(
			&data.ID, &data.UserName, &data.Login, &data.Role, &data.Role_ID, &data.Status); err != nil {
			return nil, err
		}
		datas = append(datas, data)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return datas, nil

}

func (r *Repo) DeleteUserPermission(userId, routeId int) error {
	_, err := r.store.db.Exec(`delete from auth.permissions
								where user_id = $1 and route_id = $2`, userId, routeId)
	if err != nil {
		return err
	}
	return err
}

func (r *Repo) AddUserPermission(userId, routeId int) error {
	_, err := r.store.db.Exec(`insert into auth.permissions (user_id, route_id) values ($1, $2)`, userId, routeId)
	if err != nil {
		return err
	}
	return nil
}
