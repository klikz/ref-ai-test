package api

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
)

func (s *ServerModel) Login(c *gin.Context) {
	user := models.User{}

	if err := c.ShouldBind(&user); err != nil {
		s.Utils.SendError(c, err, "Login: Error Parsing body", "")
		return
	}

	s.Utils.DebugLogAny(user.Login)
	s.Utils.DebugLogAny(user.Password)
	if err := s.Store.Repo().FindByLogin(&user); err != nil {
		s.Utils.SendError(c, err, "Login: FindByLogin", "")
		return
	}

	if !s.Utils.ComparePassword(user.Password, user.EncryptedPassword) {
		s.Utils.SendError(c, errors.New("wrong credentials"), "Login: ComparePassword", "")
		return
	}

	if err := s.Utils.GetToken(&user); err != nil {
		s.Utils.SendError(c, err, "Login: GetToken", "")
		return
	}

	if err := s.Store.Repo().SaveToken(user.Token, user.ID); err != nil {
		s.Utils.SendError(c, err, "Login: SaveToken", "")
		return
	}

	user.Password = ""
	s.Utils.SendOK(c, user)

}

func (s *ServerModel) Create(c *gin.Context) {

	user := models.User{}

	data, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "Create: ReadBody", user)
		return
	}

	user.Login = data["login"].(string)
	user.UserName = data["name"].(string)
	if accountType, ok := data["account_type"].(string); ok && accountType != "" {
		roleID, err := s.Store.Repo().GetRoleIDByName(normalizeAccountType(accountType))
		if err != nil {
			s.Utils.SendError(c, err, "Create: GetRoleIDByName", user)
			return
		}
		user.Role_ID = roleID
	} else {
		user.Role_ID = data["role_id"].(float64)
	}
	user.Password = data["password"].(string)

	pattern := regexp.MustCompile("^[A-Za-z][A-Za-z0-9]*$")
	user.UserName = strings.TrimSpace(user.UserName)

	containsOnlyAlphabetsAndNumbers := pattern.MatchString(user.Login)

	if !containsOnlyAlphabetsAndNumbers {
		s.Utils.SendError(c, errors.New("wrong login"), "Create: wrong login", user)
		return
	}

	enc, err := s.Utils.EncryptString(user.Password)
	if err != nil {
		s.Utils.SendError(c, err, "Create: encryptString", user)
		return
	}
	user.EncryptedPassword = enc

	err = s.Store.Repo().Create(&user)
	if err != nil {
		s.Utils.SendError(c, err, "Create: Create", user)
		return
	}

	s.Utils.SendOK(c, "ok")

}

func (s *ServerModel) DeleteUser(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "UserDelete: ReadBody", "")
		return
	}
	var user_id int

	if val, exists := jsonMap["user_id"]; exists && val != nil {
		s.Utils.DebugLogAny("val: ", val)
		if num, ok := val.(float64); ok {
			id := int(num)
			s.Utils.DebugLogAny("id: ", id)
			user_id = id
		}
	}
	s.Utils.DebugLogAny("user_id: ", user_id)
	err = s.Store.Repo().DeleteUser(user_id)
	if err != nil {
		s.Utils.SendError(c, err, "DeleteUser: DeleteUser", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) UpdateUser(c *gin.Context) {
	data, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "UpdateUser: ReadBody", "")
		return
	}

	user := models.User{}
	isUpdatePassword := data["is_password_update"].(bool)
	user.ID = int(data["id"].(float64))

	if isUpdatePassword {
		user.Password = data["password"].(string)

		enc, err := s.Utils.EncryptString(user.Password)
		if err != nil {
			s.Utils.SendError(c, err, "UpdateUser: encryptString", user)
			return
		}
		user.EncryptedPassword = enc
		err = s.Store.Repo().UpdatePassword(&user)
		if err != nil {
			s.Utils.SendError(c, err, "UpdateUser: UpdatePassword", user)
			return
		}
	} else {
		user.Login = data["login"].(string)
		user.UserName = data["name"].(string)
		if accountType, ok := data["account_type"].(string); ok && accountType != "" {
			roleID, err := s.Store.Repo().GetRoleIDByName(normalizeAccountType(accountType))
			if err != nil {
				s.Utils.SendError(c, err, "UpdateUser: GetRoleIDByName", "")
				return
			}
			user.Role_ID = roleID
		}

		pattern := regexp.MustCompile("^[a-zA-Z0-9]+$")
		containsOnlyAlphabetsAndNumbers := pattern.MatchString(user.Login)
		if !containsOnlyAlphabetsAndNumbers {
			s.Utils.SendError(c, errors.New("wrong login"), "UpdateUser: wrong login", "")
			return
		}

		err = s.Store.Repo().UpdateUserInfo(&user)
		if err != nil {
			s.Utils.SendError(c, err, "UpdateUserInfo: UpdateUserInfo", user)
			return
		}

	}

	s.Utils.SendOK(c, "ok")

}

func normalizeAccountType(accountType string) string {
	if strings.EqualFold(strings.TrimSpace(accountType), "admin") {
		return "admin"
	}
	return "user"
}

func (s *ServerModel) GetAllUsers(c *gin.Context) {

	data, err := s.Store.Repo().GetAllUsers()
	if err != nil {
		s.Utils.SendError(c, err, "GetAllUsers", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) GetUserById(c *gin.Context) {
	id := c.Param("id")
	user := models.User{}
	user.ID, _ = strconv.Atoi(id)
	err := s.Store.Repo().GetUserById(&user)
	if err != nil {
		s.Utils.SendError(c, err, "GetUserById", "")
		return
	}

	permissions, err := s.Store.Repo().GetUserPermissionsById(user.ID)
	if err != nil {
		s.Utils.SendError(c, err, "GetUserPermissionsById", "")
		return
	}

	type UserData struct {
		UserInfo    models.User          `json:"user_info"`
		Permissions []models.Permissions `json:"permissions"`
	}

	var data UserData
	data.UserInfo = user
	data.Permissions = permissions.([]models.Permissions)
	s.Utils.DebugLogAny("get all users")
	s.Utils.SendOK(c, data)

}

func (s *ServerModel) ChangeUserPermission(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		s.Utils.SendError(c, err, "ChangeUserPermission: ID error", "")
		return
	}
	data, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ChangeUserPermission: ReadBody", "")
		return
	}

	routeId := int(data["route_id"].(float64))
	status := data["status"].(bool)

	if status {
		err = s.Store.Repo().AddUserPermission(userId, routeId)
		if err != nil {
			s.Utils.SendError(c, err, "ChangeUserPermission: AddUserPermission", "")
			return
		}
	} else {
		err = s.Store.Repo().DeleteUserPermission(userId, routeId)
		if err != nil {
			s.Utils.SendError(c, err, "ChangeUserPermission: DeleteUserPermission", "")
			return
		}
	}
	s.Utils.SendOK(c, nil)
}
