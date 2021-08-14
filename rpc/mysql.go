package rpc

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var globalDBMap *sync.Map

func init() {
	globalDBMap = new(sync.Map)
}

var ErrConnNil = fmt.Errorf("conn is nil")

type DBInfo struct {
	*gorm.DB
	Conf *dbConfig
}

func loadDB(key string) *DBInfo {
	i, ok := globalDBMap.Load(key)
	if !ok {
		return nil
	}
	if dbInfo, ok := i.(*DBInfo); ok {
		return dbInfo
	}
	return nil
}

func initDBClient(s *Server) {
	for _, dbCfg := range s.cfg.DBClients {
		cfg := dbCfg
		info, err := initDB(&cfg)
		if err != nil {
			s.Log.Error(err.Error())
			continue
		}
		globalDBMap.Store(cfg.ServiceName, info)
	}
}

func initDB(cfg *dbConfig) (*DBInfo, error) {
	datasource := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		cfg.Username, cfg.Password, "tcp", cfg.Host, cfg.Port, cfg.Database)
	var gormConfig *gorm.Config
	if cfg.EnableLog {
		gormConfig = &gorm.Config{}
	} else {
		gormConfig = &gorm.Config{
			Logger: gormLogger.Discard,
		}
	}
	db, err := gorm.Open(mysql.Open(datasource), gormConfig)
	if err != nil {
		return &DBInfo{DB: db, Conf: cfg}, fmt.Errorf("open mysql [%s] %s:%d error: %s", cfg.ServiceName, cfg.Host, cfg.Port, err.Error())
	}

	return &DBInfo{DB: db, Conf: cfg}, nil
}

type DBConn struct {
	*gorm.DB
	ctx   context.Context
	Error error
}

func NewDBConn(ctx context.Context, serverName string) *DBConn {
	dbInfo := loadDB(serverName)
	if dbInfo == nil {
		return &DBConn{
			Error: fmt.Errorf("no such db conn with service name [%s]", serverName),
		}
	}

	pinger, ok := dbInfo.DB.ConnPool.(interface{ Ping() error })
	if !ok {
		return &DBConn{
			Error: fmt.Errorf("db conn [%s] get pinger failed", serverName),
		}
	}
	err := pinger.Ping()
	if err == nil {
		// 链接正常，返回
		return &DBConn{
			DB:  dbInfo.DB.WithContext(ctx),
			ctx: ctx,
		}
	}
	// 链接不正常，重新初始化
	// 最新版本的gorm已经去掉了close()方法
	// 重新初始化时如果报错则不fatal
	dbInfo, err = initDB(dbInfo.Conf)
	if err != nil {
		return &DBConn{
			DB:    dbInfo.DB,
			Error: fmt.Errorf("reinit db conn [%s] failed: %v", serverName, dbInfo.Error),
		}
	}
	globalDBMap.Store(serverName, dbInfo)
	return &DBConn{
		DB:  dbInfo.DB.WithContext(ctx),
		ctx: ctx,
	}
}

func (c *DBConn) WithContext(ctx context.Context) *DBConn {
	if c == nil {
		return &DBConn{
			Error: ErrConnNil,
		}
	}
	if c.Error != nil {
		return c
	}
	c.ctx = ctx
	return c
}

func (c *DBConn) RawAndScan(dest interface{}, sql string, values ...interface{}) error {
	if c == nil {
		return ErrConnNil
	}
	if c.Error != nil {
		return c.Error
	}
	return c.Raw(sql, values...).Scan(dest).Error
}

// deprecated
func DoMysqlRawWithScan(ctx context.Context, serverName string, dest interface{}, sql string, values ...interface{}) error {
	return NewDBConn(ctx, serverName).RawAndScan(dest, sql, values...)
}

// deprecated
func DoMysql(ctx context.Context, serverName string) *gorm.DB {
	db := NewDBConn(ctx, serverName)
	return db.DB
}

// deprecated
func RawAndScan(db *gorm.DB, dest interface{}, sql string, values ...interface{}) error {
	if db == nil {
		return fmt.Errorf("invalid db conn")
	}
	return db.Raw(sql, values...).Scan(dest).Error
}
