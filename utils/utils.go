package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

var printerHTTPClient = &http.Client{Timeout: 5 * time.Second}

// var Secret_key = []byte("Some123SecretKeyPremier1")

type UtilsStruct struct {
	Store  store.Store
	Logger *zerolog.Logger
	C      *gin.Context
	Key    []byte
}

func NewUtils(store *store.Store, logger *zerolog.Logger, key []byte) *UtilsStruct {
	return &UtilsStruct{
		Store:  *store,
		Logger: logger,
		Key:    key,
	}
}

func (u *UtilsStruct) SendOK(c *gin.Context, data any) {
	resp := models.Responce{}
	resp.Result = "ok"
	resp.Data = data
	c.JSON(http.StatusOK, resp)
}

func (u *UtilsStruct) SendError(c *gin.Context, err error, route string, data any) {
	resp := models.Responce{}
	message := formatErrorMessage(err)
	logEvent := u.Logger.Error().Str("route", route)
	if c != nil {
		if login, ok := c.Get("user_login"); ok {
			if s, ok := login.(string); ok && s != "" {
				logEvent = logEvent.Str("login", s)
			}
		}
		if userID := c.GetInt("user_id"); userID > 0 {
			logEvent = logEvent.Int("user_id", userID)
		}
		if path := c.FullPath(); path != "" {
			logEvent = logEvent.Str("http_path", path)
		}
		if method := c.Request.Method; method != "" {
			logEvent = logEvent.Str("http_method", method)
		}
	}
	if data != nil && data != "" {
		logEvent = logEvent.Interface("data", data)
	}
	EnrichErrorLog(logEvent, err)
	logEvent.Msg(FormatDBError(err))
	resp.Result = "error"
	resp.Error = message
	resp.Data = data
	c.JSON(http.StatusBadRequest, resp)
	c.Abort()
}

func formatErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		key := duplicateKeyName(pqErr)
		if key == "" {
			key = "unique key"
		}
		return key + ": bunday ma'lumot kiritilgan"
	}

	message := err.Error()
	if strings.Contains(message, "duplicate key") || strings.Contains(message, "повторяющееся значение") {
		key := duplicateKeyFromDetail(message)
		if key == "" {
			key = "unique key"
		}
		return key + ": bunday ma'lumot kiritilgan"
	}
	return message
}

func duplicateKeyName(err *pq.Error) string {
	if key := duplicateKeyFromDetail(err.Detail); key != "" {
		return key
	}
	return strings.TrimSpace(err.Constraint)
}

func duplicateKeyFromDetail(detail string) string {
	start := strings.Index(detail, "Key (")
	if start == -1 {
		return ""
	}
	start += len("Key (")
	end := strings.Index(detail[start:], ")")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(detail[start : start+end])
}

func (u *UtilsStruct) DebugLogAny(data ...any) {
	if u == nil || u.Logger == nil {
		return
	}
	u.Logger.Debug().Msg(fmt.Sprintf("%v", data))
}

func (u *UtilsStruct) ProdLogAny(data ...any) {
	u.Logger.Info().Msg(fmt.Sprintf("%v", data))
}

// PrintLogWarn — printer muammolari (spool stuck, retry, GDI mismatch). Info+ level'da ko'rinadi.
func (u *UtilsStruct) PrintLogWarn(msg string, fields map[string]any) {
	u.printLogAt(zerolog.WarnLevel, msg, fields)
}

// PrintLogError — yakuniy print xatosi.
func (u *UtilsStruct) PrintLogError(msg string, fields map[string]any) {
	u.printLogAt(zerolog.ErrorLevel, msg, fields)
}

func (u *UtilsStruct) printLogAt(level zerolog.Level, msg string, fields map[string]any) {
	if u == nil || u.Logger == nil {
		return
	}
	ev := u.Logger.WithLevel(level).Str("component", "print_v2")
	for k, v := range fields {
		if k == "" || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			ev = ev.Str(k, t)
		case int:
			ev = ev.Int(k, t)
		case int64:
			ev = ev.Int64(k, t)
		case bool:
			ev = ev.Bool(k, t)
		case float64:
			ev = ev.Float64(k, t)
		case time.Duration:
			ev = ev.Dur(k, t)
		default:
			ev = ev.Interface(k, t)
		}
	}
	ev.Msg(msg)
}

func (u *UtilsStruct) ReadBody(c *gin.Context) (map[string]any, error) {
	bodyAsByteArray, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	jsonMap := make(map[string]any)
	err = json.Unmarshal(bodyAsByteArray, &jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

func (u *UtilsStruct) CheckRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		parsedToken, err := u.ParseToken(token)
		if err != nil {
			u.SendError(c, err, "CheckRole: ParseToken", "token error")
			return
		}
		c.Set("user_login", parsedToken.Login)

		userID := parsedToken.UserID
		if userID == 0 {
			userID, err = u.Store.Repo().GetUserID(parsedToken.Login)
			if err != nil {
				u.SendError(c, err, "CheckRole: GetUserID", "")
				return
			}
		}
		c.Set("user_id", userID)

		ok, err := u.Store.Repo().EnsureRouteAndCheckPermission(userID, c.FullPath())
		if err != nil {
			u.SendError(c, errors.New(c.FullPath()+": "+err.Error()), "CheckRole: EnsureRouteAndCheckPermission", "")
			return
		}
		if !ok {
			u.SendError(c, errors.New(c.FullPath()+": no permission"), "CheckRole", "")
			return
		}
		c.Next()
	}
}

func (u *UtilsStruct) ParseToken(tokenString string) (models.ParsedToken, error) {
	parsedToken := models.ParsedToken{}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return u.Key, nil
	})
	if err != nil {
		return parsedToken, err
	}

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

	} else {
		return parsedToken, errors.New("wrong token")
	}
	parsedToken.Login = fmt.Sprint(claims["login"])
	if uid, ok := claims["user_id"].(float64); ok {
		parsedToken.UserID = int(uid)
	}

	return parsedToken, nil
}

func (u *UtilsStruct) EncryptString(s string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (u *UtilsStruct) ComparePassword(password, encrypt string) bool {
	return bcrypt.CompareHashAndPassword([]byte(encrypt), []byte(password)) == nil
}

func (u *UtilsStruct) GetToken(user *models.User) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.UserName,
		"login":    user.Login,
		"user_id":  user.ID,
		"nbf":      time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})
	tokenString, err := token.SignedString(u.Key)
	if err != nil {
		return err
	}
	user.Token = tokenString
	return nil
}

func Base64Decode(str string) (string, error) {
	// fmt.Println("fabse64 string: ", str)
	b64data := str[strings.IndexByte(str, ',')+1:]

	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// PrintLabel is kept for compatibility; prefer PrintLabelAndWait.
func (u *UtilsStruct) PrintLabel(jsonStr []byte, printerURL string) (<-chan string, func()) {
	ch := make(chan string, 1)
	ch <- u.PrintLabelAndWait(jsonStr, printerURL)
	close(ch)
	return ch, func() {}
}
