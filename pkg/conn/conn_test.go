package conn

import (
	"strings"
	"testing"
	"time"
)

func TestValidateDbOptions(t *testing.T) {
	tests := []struct {
		name    string
		opt     *DbOptions
		wantErr string
	}{
		{name: "nil", wantErr: "nil db options"},
		{name: "missing user", opt: &DbOptions{Address: "127.0.0.1:3306", Schema: "db"}, wantErr: "db user is required"},
		{name: "missing address", opt: &DbOptions{User: "root", Schema: "db"}, wantErr: "db address is required"},
		{name: "missing schema", opt: &DbOptions{User: "root", Address: "127.0.0.1:3306"}, wantErr: "db schema is required"},
		{name: "unsupported dialect", opt: &DbOptions{User: "root", Address: "127.0.0.1:3306", Schema: "db", Dialect: "postgres"}, wantErr: "unsupported db dialect"},
		{name: "valid", opt: &DbOptions{User: "root", Address: "127.0.0.1:3306", Schema: "db"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDbOptions(tt.opt)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateDbOptions() error = %v", err)
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateDbOptions() error = %v, want substring %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestMySQLDSN(t *testing.T) {
	opt := &DbOptions{
		User:     "root",
		Password: "secret",
		Address:  "db:3306",
		Schema:   "gomicro",
	}

	got := mysqlDSN(opt, "charset=utf8&parseTime=true")
	want := "root:secret@tcp(db:3306)/gomicro?charset=utf8&parseTime=true"
	if got != want {
		t.Fatalf("mysqlDSN() = %q, want %q", got, want)
	}
}

func TestOpenSQLAppliesPoolSettings(t *testing.T) {
	db, err := open(&DbOptions{
		User:    "root",
		Address: "db:3306",
		Schema:  "gomicro",
		ConnPool: &DbPoolSettings{
			MaxIdleConns:    3,
			MaxOpenConns:    5,
			MaxLifetime:     time.Minute,
			MaxIdleLifetime: 2 * time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("open() error = %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 5 {
		t.Fatalf("MaxOpenConnections = %d, want 5", stats.MaxOpenConnections)
	}
}

func TestOpenGormRejectsUnsupportedDialect(t *testing.T) {
	_, err := openGorm(&DbOptions{
		User:    "root",
		Address: "db:3306",
		Schema:  "gomicro",
		Dialect: "postgres",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported db dialect") {
		t.Fatalf("openGorm() error = %v, want unsupported dialect", err)
	}
}
