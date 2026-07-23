package api

import (
	"errors"
	"os"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
	"github.com/rs/zerolog"
)

type ServerModel struct {
	Router            *gin.Engine
	Logger            *zerolog.Logger
	Store             store.Store
	Utils             *utils.UtilsStruct
	CurresntT3Serial  string
	CurrentT3V2Serial string
	T3SerialHub       *T3SerialHub
}

func StartSrv(port string, mode string, logger *zerolog.Logger, store store.Store) error {
	// gin.SetMode(gin.ReleaseMode)
	switch mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	s := &ServerModel{
		Router:            gin.New(),
		Logger:            logger,
		Store:             store,
		Utils:             utils.NewUtils(&store, logger, []byte(os.Getenv("SECRET_KEY"))),
		CurresntT3Serial:  "",
		CurrentT3V2Serial: "",
		T3SerialHub:       NewT3SerialHub(),
	}

	s.configureRouter()

	s.Logger.Info().Msg("Server starting on port " + port)
	err := s.Router.Run(":" + port)

	if err != nil {
		return err
	}

	return nil

}

func (s *ServerModel) configureRouter() {
	s.Router.SetTrustedProxies([]string{"localhost"})
	s.Router.Use(CORSMiddleware())
	s.Router.GET(t3SerialWSRoute, s.LinesT3V2SerialCurrentWS)
	s.Router.Use(gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".mp3", ".woff", ".woff2"}),
	))
	s.Router.Use(staticCacheMiddleware())

	// Serve static files from the React app build directory
	s.Router.Static("/assets", "./web/build/assets")
	s.Router.Static("/uploads", "./uploads")
	s.Router.StaticFile("/favicon.ico", "./web/build/favicon.ico")
	s.Router.StaticFile("/p.ico", "./web/build/p.ico")
	s.Router.StaticFile("/beep_ok.mp3", "./web/build/beep_ok.mp3")
	s.Router.StaticFile("/alert.mp3", "./web/build/alert.mp3")
	// s.Router.Static("/files", "./files")
	// s.Router.Static("/static", "./web/build/static")

	// s.Router.Use(fs.Serve("/", static.LocalFile("./client/build", true)))
	s.Router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "POST" {
			if c.FullPath() == "" {
				s.Utils.SendError(c, errors.New("wrong route"), "wrong route", "")
				return
			}

			id, err := s.Store.Repo().GetRouteID(c.FullPath())
			if err != nil || id == 0 {
				s.Utils.SendError(c, errors.New("bad request"), "bad request", "")
				return
			}
		} else {
			reqPath := c.Request.URL.Path
			if reqPath != "/" && !strings.HasPrefix(reqPath, "/api") {
				candidate := "./web/build" + reqPath
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					c.File(candidate)
					return
				}
			}
			c.File("./web/build/index.html")
		}
	})
	s.Routes()

	// Handle all other routes by serving index.html
}

func staticCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (s *ServerModel) Ping(c *gin.Context) {

	data, err := s.Store.Repo().GetAllUsers()
	if err != nil {
		s.Logger.Error().Msg(err.Error())
	}
	c.JSON(200, data)
}
