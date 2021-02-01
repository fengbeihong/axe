package rpc

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var globalDBMap map[string]*DBInfo

func init() {
	globalDBMap = make(map[string]*DBInfo)
}

type DBInfo struct {
	*gorm.DB
}

func initDBClient(cfg *Config) {
	for _, dbCfg := range cfg.DBClients {
		info, err := initDB(&dbCfg)
		if err != nil {
			log.Fatalf(err.Error())
		}
		globalDBMap[dbCfg.ServiceName] = info
	}
}

func initDB(cfg *dbConfig) (*DBInfo, error) {
	datasource := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		cfg.Username, cfg.Password, "tcp", cfg.Host, cfg.Port, cfg.Database)
	db, err := gorm.Open(mysql.Open(datasource), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql [%s] %s:%d error: %s", cfg.ServiceName, cfg.Host, cfg.Port, err.Error())
	}

	return &DBInfo{DB: db}, nil
}

func DoMysqlRawWithScan(ctx context.Context, serverName string, dest interface{}, sql string, values ...interface{}) error {
	return globalDBMap[serverName].DB.WithContext(ctx).Raw(sql, values...).Scan(dest).Error
}

func DoMysql(ctx context.Context, serverName string) *gorm.DB {
	return globalDBMap[serverName].DB.WithContext(ctx)
}
