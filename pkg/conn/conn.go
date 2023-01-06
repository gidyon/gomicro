package conn

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm/logger"

	"gorm.io/gorm"

	// Imports mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// DbPoolSettings contains options for customizing the connection pool
type DbPoolSettings struct {
	MaxIdleConns    uint
	MaxOpenConns    uint
	MaxLifetime     time.Duration
	MaxIdleLifetime time.Duration
}

// DbOptions contains parameters for connecting to a SQL database
type DbOptions struct {
	Name     string
	Dialect  string
	Address  string
	User     string
	Password string
	Schema   string
	ConnPool *DbPoolSettings
}

// OpenGorm open a connection to sql database using gorm orm
func OpenGorm(opt *DbOptions) (*gorm.DB, error) {
	return openGorm(opt)
}

// opens a connection to SQL database returning gorm database client
func openGorm(opt *DbOptions) (*gorm.DB, error) {
	// Options should not be nil
	if opt == nil {
		return nil, errors.New("nil db options not allowed")
	}

	// add MySQL driver specific parameter to parse date/time
	param := "charset=utf8&parseTime=true"

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		opt.User,
		opt.Password,
		opt.Address,
		opt.Schema,
		param,
	)

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

// OpenSql open a connection to sql database
func OpenSql(opt *DbOptions) (*sql.DB, error) {
	return open(opt)
}

// opens a connection to the SQL database returning sql database client
func open(opt *DbOptions) (*sql.DB, error) {
	// Options should not be nil
	if opt == nil {
		return nil, errors.New("nil db options not allowed")
	}

	// add MySQL driver specific parameter to parse date/time
	param := "charset=utf8&parseTime=true"

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		opt.User,
		opt.Password,
		opt.Address,
		opt.Schema,
		param,
	)

	dialect := func() string {
		if opt.Dialect == "" {
			return "mysql"
		}
		return opt.Dialect
	}()

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
	}

	return sqlDB, nil
}
