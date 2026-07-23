package main

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"time"

	"github.com/klikz/api_v3/internal/api"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/rs/zerolog"

	"github.com/joho/godotenv"
)

func main() {
	//init config

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	//init logger

	file, err := os.OpenFile("logger.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	multi := zerolog.MultiLevelWriter(consoleWriter, file)

	log_level := os.Getenv("LOG_LEVEL")
	if log_level == "debug" {
		zerolog.SetGlobalLevel(0)
	} else {
		zerolog.SetGlobalLevel(1)
	}

	logger := zerolog.New(multi).With().Timestamp().Logger()

	//init database

	connString, err := utils.DBConnString()
	if err != nil {
		logger.Fatal().Msg(err.Error())
	}
	db, err := sql.Open("postgres", connString)
	if err != nil {
		logger.Fatal().Msg("Cannot connect to database! Dying..." + err.Error())
	}
	defer db.Close()

	db.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", 25))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 10))
	db.SetConnMaxLifetime(time.Duration(envInt("DB_CONN_MAX_LIFETIME_MIN", 30)) * time.Minute)

	storeInst := store.New(db)

	err = db.Ping()
	if err != nil {
		logger.Fatal().Msg("err: " + err.Error())
	}
	logger.Info().Msg("Database connection established")

	labDB, err := sql.Open("sqlserver", utils.VtmMSSQLConnString())
	if err != nil {
		logger.Warn().Err(err).Str("target", utils.VtmMSSQLHostForLog()).Msg("VTM MSSQL open failed; lab API will be unavailable")
	} else {
		labDB.SetMaxOpenConns(10)
		labDB.SetMaxIdleConns(2)
		labDB.SetConnMaxLifetime(30 * time.Minute)
		timeoutSec := envInt("VTM_MSSQL_TIMEOUT_SEC", 10)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
		pingErr := labDB.PingContext(ctx)
		cancel()
		if pingErr != nil {
			logger.Warn().Err(pingErr).Str("target", utils.VtmMSSQLHostForLog()).Msg("VTM MSSQL ping failed; lab API will be unavailable")
			_ = labDB.Close()
		} else {
			storeInst.SetLabDB(labDB)
			defer labDB.Close()
			logger.Info().Str("target", utils.VtmMSSQLHostForLog()).Msg("VTM MSSQL connection established")
		}
	}

	utils.InitPrintPool(envInt("PRINT_MAX_WORKERS", 8))

	//run server
	err = api.StartSrv(os.Getenv("PORT"), log_level, &logger, *storeInst)
	if err != nil {
		logger.Error().Msg(err.Error())
	}
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
