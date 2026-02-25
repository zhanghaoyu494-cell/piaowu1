package main

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type CustomerService struct {
	CsID          string     `gorm:"column:cs_id;type:varchar(32);primaryKey"`
	CsName        string     `gorm:"column:cs_name;type:varchar(64);not null"`
	DeptID        string     `gorm:"column:dept_id;type:varchar(32);not null"`
	TeamID        string     `gorm:"column:team_id;type:varchar(32)"`
	Status        int8       `gorm:"column:status;type:tinyint(1);not null"`
	CurrentStatus int8       `gorm:"column:current_status;type:tinyint(1);not null"`
	IsOnline      int8       `gorm:"column:is_online;type:tinyint(1);not null;default:0"`
	LastHeartbeat *time.Time `gorm:"column:last_heartbeat"`
	Role          int8       `gorm:"column:role;type:tinyint(1);not null;default:0"`
	CreateTime    time.Time  `gorm:"column:create_time;not null"`
	UpdateTime    time.Time  `gorm:"column:update_time;not null"`
}

func (CustomerService) TableName() string {
	return "t_customer_service"
}

func main() {
	// 与 customer/config/config.yaml 保持一致
	dsn := "root:Zhyzhy666888@tcp(121.5.9.239:3306)/ccc?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	now := time.Now()
	agent := CustomerService{
		CsID:          "CS999",
		CsName:        "值班客服",
		DeptID:        "D1",
		TeamID:        "T0",
		Status:        1,
		CurrentStatus: 0,
		IsOnline:      1,
		LastHeartbeat: &now,
		Role:          0,
		CreateTime:    now,
		UpdateTime:    now,
	}

	var existing CustomerService
	err = db.Where("cs_id = ?", agent.CsID).First(&existing).Error
	if err == nil {
		// 更新为在线、刷新心跳
		err = db.Model(&existing).Updates(map[string]interface{}{
			"cs_name":        agent.CsName,
			"dept_id":        agent.DeptID,
			"team_id":        agent.TeamID,
			"status":         1,
			"current_status": 0,
			"is_online":      1,
			"last_heartbeat": agent.LastHeartbeat,
			"update_time":    now,
		}).Error
		if err != nil {
			log.Fatalf("update fallback agent failed: %v", err)
		}
		log.Printf("fallback agent updated online with heartbeat: %s", now.Format("2006-01-02 15:04:05"))
		return
	}

	if err != nil && err == gorm.ErrRecordNotFound {
		if err = db.Create(&agent).Error; err != nil {
			log.Fatalf("create fallback agent failed: %v", err)
		}
		log.Printf("fallback agent created and set online")
		return
	}

	log.Fatalf("lookup fallback agent failed: %v", err)
}
