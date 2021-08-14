package rpc

import (
	"context"
	"testing"
)

var mysqlTestConfig = &Server{
	cfg: &Config{
		DBClients: []dbConfig{
			{
				ServiceName: "test",
				Host:        "127.0.0.1",
				Port:        3306,
				Username:    "root",
				Password:    "aaaaaaaa",
				Database:    "test",
			},
		},
	},
	Log: defaultLogger(),
}

func TestMysqlConn(t *testing.T) {
	initDBClient(mysqlTestConfig)

	var dest []struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}

	conn := NewDBConn(context.Background(), "test")
	if pinger, ok := conn.DB.ConnPool.(interface{ Ping() error }); ok {
		err := pinger.Ping()
		if err != nil {
			t.Fatal("TestMysqlConn new db conn failed, error: ", err.Error())
		}
	}

	err := conn.RawAndScan(&dest, "select id, name from test")
	if err != nil {
		t.Fatal("TestMysqlConn failed, error: ", err.Error())
	}
	t.Log("TestMysqlConn pass, example data: ", dest)
}

func TestMysqlConnConcurrent(t *testing.T) {
	initDBClient(mysqlTestConfig)

	conn := NewDBConn(context.Background(), "test")
	for i := 0; i < 30; i++ {
		var dest []struct {
			ID   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		err := conn.RawAndScan(&dest, "select id, name from test")
		if err != nil {
			t.Fatal("TestMysqlConnConcurrent failed, error: ", err.Error())
		}
	}
}
