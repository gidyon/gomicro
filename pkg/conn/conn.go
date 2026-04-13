package conn

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm/logger"

	"gorm.io/gorm"

	// Imports mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// DbPoolSettings controls SQL connection pool sizing and lifetimes.
type DbPoolSettings struct {
	MaxIdleConns    uint
	MaxOpenConns    uint
	MaxLifetime     time.Duration
	MaxIdleLifetime time.Duration
}

// DbOptions contains parameters for connecting to a SQL database.
// Only MySQL is currently supported.
type DbOptions struct {
	Name     string
	Dialect  string
	Address  string
	User     string
	Password string
	Schema   string
	ConnPool *DbPoolSettings
}

const defaultSQLDialect = "mysql"

// OpenGorm opens a GORM connection using the configured MySQL settings.
func OpenGorm(opt *DbOptions) (*gorm.DB, error) {
	return openGorm(opt)
}

// opens a connection to SQL database returning gorm database client
func openGorm(opt *DbOptions) (*gorm.DB, error) {
	if err := validateDbOptions(opt); err != nil {
		return nil, err
	}

	dsn := mysqlDSN(opt, "charset=utf8mb4&parseTime=True")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("(GORM) failed to open connection to mysql database [name=%s] [address=%s] : %v ", opt.Name, opt.Address, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if opt.ConnPool != nil {
		if opt.ConnPool.MaxIdleConns != 0 {
			sqlDB.SetMaxIdleConns(int(opt.ConnPool.MaxIdleConns))
		}
		if opt.ConnPool.MaxOpenConns != 0 {
			sqlDB.SetMaxOpenConns(int(opt.ConnPool.MaxOpenConns))
		}
		if opt.ConnPool.MaxLifetime != 0 {
			sqlDB.SetConnMaxLifetime(opt.ConnPool.MaxLifetime)
		}
		if opt.ConnPool.MaxIdleLifetime != 0 {
			sqlDB.SetConnMaxIdleTime(opt.ConnPool.MaxIdleLifetime)
		}
	}

	return db, nil
}

// OpenSql opens a database/sql connection using the configured MySQL settings.
func OpenSql(opt *DbOptions) (*sql.DB, error) {
	return open(opt)
}

// opens a connection to the SQL database returning sql database client
func open(opt *DbOptions) (*sql.DB, error) {
	if err := validateDbOptions(opt); err != nil {
		return nil, err
	}

	dsn := mysqlDSN(opt, "charset=utf8&parseTime=true")
	dialect := sqlDialect(opt)

	sqlDB, err := sql.Open(dialect, dsn)
	if err != nil {
		return nil, fmt.Errorf("(SQL) failed to open connection to mysql database [name=%s] [address=%s]: %v", opt.Name, opt.Address, err)
	}

	if opt.ConnPool != nil {
		if opt.ConnPool.MaxIdleConns != 0 {
			sqlDB.SetMaxIdleConns(int(opt.ConnPool.MaxIdleConns))
		}
		if opt.ConnPool.MaxOpenConns != 0 {
			sqlDB.SetMaxOpenConns(int(opt.ConnPool.MaxOpenConns))
		}
		if opt.ConnPool.MaxLifetime != 0 {
			sqlDB.SetConnMaxLifetime(opt.ConnPool.MaxLifetime)
		}
		if opt.ConnPool.MaxIdleLifetime != 0 {
			sqlDB.SetConnMaxIdleTime(opt.ConnPool.MaxIdleLifetime)
		}
	}

	return sqlDB, nil
}

func validateDbOptions(opt *DbOptions) error {
	if opt == nil {
		return errors.New("nil db options not allowed")
	}
	if strings.TrimSpace(opt.User) == "" {
		return errors.New("db user is required")
	}
	if strings.TrimSpace(opt.Address) == "" {
		return errors.New("db address is required")
	}
	if strings.TrimSpace(opt.Schema) == "" {
		return errors.New("db schema is required")
	}
	if dialect := sqlDialect(opt); dialect != defaultSQLDialect {
		return fmt.Errorf("unsupported db dialect %q", dialect)
	}
	return nil
}

func sqlDialect(opt *DbOptions) string {
	if opt == nil || strings.TrimSpace(opt.Dialect) == "" {
		return defaultSQLDialect
	}
	return strings.TrimSpace(opt.Dialect)
}

func mysqlDSN(opt *DbOptions, params string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		opt.User,
		opt.Password,
		opt.Address,
		opt.Schema,
		params,
	)
}
