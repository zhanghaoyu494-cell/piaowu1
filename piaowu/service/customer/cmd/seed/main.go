package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"example_shop/service/customer/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Database connection string from config.yaml
	dsn := "root:Zhyzhy666888@tcp(localhost:3306)/ccc?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	var shiftCount, csCount, schedCount, convCount, msgCount, ltCount int64
	db.Model(&model.ShiftConfig{}).Count(&shiftCount)
	db.Model(&model.CustomerService{}).Count(&csCount)
	db.Model(&model.Schedule{}).Count(&schedCount)
	db.Model(&model.Conversation{}).Count(&convCount)
	db.Model(&model.ConvMessage{}).Count(&msgCount)
	db.Model(&model.LeaveTransfer{}).Count(&ltCount)

	log.Printf("Current Counts: Shift=%d, CS=%d, Sched=%d, Conv=%d, Msg=%d, LT=%d",
		shiftCount, csCount, schedCount, convCount, msgCount, ltCount)

	// Always ensure test CS names are correct
	updateTestCS(db)

	// Only run seeding if counts are low
	if shiftCount < 10 {
		seedShiftConfig(db)
	}
	if csCount < 10 {
		csList := seedCustomerService(db)
		if schedCount < 10 {
			seedSchedule(db, csList)
		}
		if convCount < 200 {
			log.Println("Starting seedConversation...")
			seedConversation(db, csList)
		}
		if ltCount < 10 {
			seedLeaveTransfer(db, csList)
		}
	} else {
		// If CS exists, we need to get IDs for other seeds
		var csList []string
		db.Model(&model.CustomerService{}).Pluck("cs_id", &csList)
		if schedCount < 10 {
			seedSchedule(db, csList)
		}
		if convCount < 200 {
			seedConversation(db, csList)
		}
		if ltCount < 10 {
			seedLeaveTransfer(db, csList)
		}
	}

	if convCount == 0 { // Explicitly retry conversation if 0
		var csList []string
		db.Model(&model.CustomerService{}).Pluck("cs_id", &csList)
		seedConversation(db, csList)
	}

	seedConvCategory(db)
	seedConvTag(db)
	seedQuickReply(db)

	log.Println("Seeding check/update completed!")
}

func updateTestCS(db *gorm.DB) {
	testCS := []struct {
		ID   string
		Name string
	}{
		{"CS001", "测试客服1"},
		{"CS002", "测试客服2"},
		{"CS003", "测试客服3"},
	}
	for _, tc := range testCS {
		db.Model(&model.CustomerService{}).Where("cs_id = ?", tc.ID).Update("cs_name", tc.Name)
	}
	log.Println("Updated Test CS Names")
}

func seedShiftConfig(db *gorm.DB) {
	shifts := []map[string]interface{}{
		{"shift_name": "早班", "start_time": "1970-01-01 08:00:00", "end_time": "1970-01-01 16:00:00", "min_staff": 3, "is_holiday": 0, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "中班", "start_time": "1970-01-01 16:00:00", "end_time": "1970-01-01 23:59:59", "min_staff": 3, "is_holiday": 0, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()}, // 24:00:00 is not valid datetime
		{"shift_name": "晚班", "start_time": "1970-01-01 00:00:00", "end_time": "1970-01-01 08:00:00", "min_staff": 2, "is_holiday": 0, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "周末值班", "start_time": "1970-01-01 09:00:00", "end_time": "1970-01-01 18:00:00", "min_staff": 2, "is_holiday": 1, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "高峰支援", "start_time": "1970-01-01 10:00:00", "end_time": "1970-01-01 14:00:00", "min_staff": 5, "is_holiday": 0, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "节假日早班", "start_time": "1970-01-01 08:00:00", "end_time": "1970-01-01 16:00:00", "min_staff": 2, "is_holiday": 1, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "节假日晚班", "start_time": "1970-01-01 16:00:00", "end_time": "1970-01-01 23:59:59", "min_staff": 2, "is_holiday": 1, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "夜间值守", "start_time": "1970-01-01 00:00:00", "end_time": "1970-01-01 08:00:00", "min_staff": 1, "is_holiday": 1, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "午间轮休", "start_time": "1970-01-01 11:00:00", "end_time": "1970-01-01 13:00:00", "min_staff": 2, "is_holiday": 0, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
		{"shift_name": "晚间支援", "start_time": "1970-01-01 18:00:00", "end_time": "1970-01-01 22:00:00", "min_staff": 3, "is_holiday": 0, "create_by": "ADMIN", "create_time": time.Now(), "update_time": time.Now()},
	}

	for _, s := range shifts {
		// Using FirstOrCreate with map condition to find existing, and map attrs to create
		var count int64
		db.Model(&model.ShiftConfig{}).Where("shift_name = ?", s["shift_name"]).Count(&count)
		if count == 0 {
			db.Model(&model.ShiftConfig{}).Create(s)
		}
	}
	log.Println("Seeded ShiftConfig")
}

func seedCustomerService(db *gorm.DB) []string {
	csData := []model.CustomerService{
		{CsID: "CS001", CsName: "张伟", DeptID: "DEPT001", TeamID: "TEAM001", SkillTags: "售后,技术", Status: 1, CurrentStatus: 1},
		{CsID: "CS002", CsName: "李娜", DeptID: "DEPT001", TeamID: "TEAM001", SkillTags: "售前,咨询", Status: 1, CurrentStatus: 1},
		{CsID: "CS003", CsName: "王强", DeptID: "DEPT001", TeamID: "TEAM002", SkillTags: "投诉,处理", Status: 1, CurrentStatus: 1},
		{CsID: "CS004", CsName: "赵敏", DeptID: "DEPT002", TeamID: "TEAM003", SkillTags: "技术,高级", Status: 1, CurrentStatus: 1},
		{CsID: "CS005", CsName: "孙杰", DeptID: "DEPT002", TeamID: "TEAM003", SkillTags: "售后,维修", Status: 1, CurrentStatus: 1},
		{CsID: "CS006", CsName: "周婷", DeptID: "DEPT002", TeamID: "TEAM004", SkillTags: "售前,引导", Status: 1, CurrentStatus: 1},
		{CsID: "CS007", CsName: "吴刚", DeptID: "DEPT003", TeamID: "TEAM005", SkillTags: "VIP,专属", Status: 1, CurrentStatus: 1},
		{CsID: "CS008", CsName: "郑丽", DeptID: "DEPT003", TeamID: "TEAM005", SkillTags: "VIP,英语", Status: 1, CurrentStatus: 1},
		{CsID: "CS009", CsName: "陈勇", DeptID: "DEPT003", TeamID: "TEAM006", SkillTags: "投诉,加急", Status: 1, CurrentStatus: 1},
		{CsID: "CS010", CsName: "刘芳", DeptID: "DEPT001", TeamID: "TEAM002", SkillTags: "普通,咨询", Status: 1, CurrentStatus: 1},
		{CsID: "CS011", CsName: "林涛", DeptID: "DEPT001", TeamID: "TEAM001", SkillTags: "技术,网络", Status: 1, CurrentStatus: 1},
		{CsID: "CS012", CsName: "何静", DeptID: "DEPT002", TeamID: "TEAM003", SkillTags: "售后,退换", Status: 1, CurrentStatus: 1},
	}

	var ids []string
	for _, cs := range csData {
		cs.CreateTime = time.Now()
		cs.UpdateTime = time.Now()
		db.FirstOrCreate(&cs, model.CustomerService{CsID: cs.CsID})
		ids = append(ids, cs.CsID)
	}
	log.Println("Seeded CustomerService")
	return ids
}

func seedConvCategory(db *gorm.DB) {
	cats := []model.ConvCategory{
		{CategoryName: "售前咨询", SortNo: 1, CreateBy: "ADMIN"},
		{CategoryName: "售后服务", SortNo: 2, CreateBy: "ADMIN"},
		{CategoryName: "技术支持", SortNo: 3, CreateBy: "ADMIN"},
		{CategoryName: "投诉建议", SortNo: 4, CreateBy: "ADMIN"},
		{CategoryName: "订单查询", SortNo: 5, CreateBy: "ADMIN"},
		{CategoryName: "退换货", SortNo: 6, CreateBy: "ADMIN"},
		{CategoryName: "活动咨询", SortNo: 7, CreateBy: "ADMIN"},
		{CategoryName: "账号问题", SortNo: 8, CreateBy: "ADMIN"},
		{CategoryName: "支付问题", SortNo: 9, CreateBy: "ADMIN"},
		{CategoryName: "其他", SortNo: 10, CreateBy: "ADMIN"},
	}
	for _, c := range cats {
		c.CreateTime = time.Now()
		c.UpdateTime = time.Now()
		db.FirstOrCreate(&c, model.ConvCategory{CategoryName: c.CategoryName})
	}
	log.Println("Seeded ConvCategory")
}

func seedConvTag(db *gorm.DB) {
	tags := []model.ConvTag{
		{TagName: "VIP客户", TagColor: "#FF0000", SortNo: 1, CreateBy: "ADMIN"},
		{TagName: "潜在客户", TagColor: "#00FF00", SortNo: 2, CreateBy: "ADMIN"},
		{TagName: "急需处理", TagColor: "#0000FF", SortNo: 3, CreateBy: "ADMIN"},
		{TagName: "已解决", TagColor: "#CCCCCC", SortNo: 4, CreateBy: "ADMIN"},
		{TagName: "待跟进", TagColor: "#FFFF00", SortNo: 5, CreateBy: "ADMIN"},
		{TagName: "恶意骚扰", TagColor: "#000000", SortNo: 6, CreateBy: "ADMIN"},
		{TagName: "多次投诉", TagColor: "#FF00FF", SortNo: 7, CreateBy: "ADMIN"},
		{TagName: "好评用户", TagColor: "#00FFFF", SortNo: 8, CreateBy: "ADMIN"},
		{TagName: "新用户", TagColor: "#888888", SortNo: 9, CreateBy: "ADMIN"},
		{TagName: "老用户", TagColor: "#444444", SortNo: 10, CreateBy: "ADMIN"},
	}
	for _, t := range tags {
		t.CreateTime = time.Now()
		t.UpdateTime = time.Now()
		db.FirstOrCreate(&t, model.ConvTag{TagName: t.TagName})
	}
	log.Println("Seeded ConvTag")
}

func seedQuickReply(db *gorm.DB) {
	replies := []model.QuickReply{
		{ReplyContent: "您好，请问有什么可以帮您？", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "稍等，我这边帮您查询一下。", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "感谢您的咨询，祝您生活愉快！", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "请提供一下您的订单号。", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "这个问题需要转接技术支持，请稍候。", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "亲，这个商品目前有优惠活动哦。", ReplyType: 1, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "退款通常在1-3个工作日内到账。", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "我们的工作时间是每天9:00-18:00。", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "非常抱歉给您带来不便。", ReplyType: 0, IsPublic: 1, CreateBy: "ADMIN"},
		{ReplyContent: "您可以关注我们的公众号获取更多资讯。", ReplyType: 1, IsPublic: 1, CreateBy: "ADMIN"},
	}
	for _, r := range replies {
		r.CreateTime = time.Now()
		r.UpdateTime = time.Now()
		db.FirstOrCreate(&r, model.QuickReply{ReplyContent: r.ReplyContent})
	}
	log.Println("Seeded QuickReply")
}

func seedSchedule(db *gorm.DB, csIDs []string) {
	var shifts []model.ShiftConfig
	db.Find(&shifts)
	if len(shifts) == 0 {
		return
	}

	startDate := time.Now()
	for i := 0; i < 14; i++ { // Next 14 days
		date := startDate.AddDate(0, 0, i).Format("2006-01-02")
		for j, csID := range csIDs {
			shift := shifts[(j+i)%len(shifts)]
			s := model.Schedule{
				CsID:         csID,
				ShiftID:      shift.ShiftID,
				ScheduleDate: date,
				Status:       0,
				CreateTime:   time.Now(),
				UpdateTime:   time.Now(),
			}
			// Use Where to avoid duplicates based on CsID and ScheduleDate
			var count int64
			db.Model(&model.Schedule{}).Where("cs_id = ? AND schedule_date = ?", csID, date).Count(&count)
			if count == 0 {
				db.Create(&s)
			}
		}
	}
	log.Println("Seeded Schedule")
}

func seedConversation(db *gorm.DB, csIDs []string) {
	users := []string{"USER001", "USER002", "USER003", "USER004", "USER005", "USER006", "USER007", "USER008", "USER009", "USER010"}
	nicknames := []string{"张三", "李四", "王五", "赵六", "钱七", "孙八", "周九", "吴十", "郑十一", "卫十二"}
	sources := []string{"Web", "App", "WeChat"}

	// Track active users to ensure uniqueness (One User -> One Active Conversation)
	activeUsers := make(map[string]bool)

	for i := 0; i < 20; i++ {
		csID := csIDs[rand.Intn(len(csIDs))]
		userIdx := rand.Intn(len(users))
		userID := users[userIdx]

		// Random status: 0-active, 1-ended
		status := int8(rand.Intn(2))

		// If user already has an active conversation, force this one to be ended (status=1)
		// This ensures A user is only active with ONE CS at a time.
		if activeUsers[userID] && status == 0 {
			status = 1 // Force ended
		}

		if status == 0 {
			activeUsers[userID] = true
		}

		convID := fmt.Sprintf("CONV%d", time.Now().UnixNano()+int64(i))
		startTime := time.Now().Add(-time.Duration(rand.Intn(100)) * time.Hour)

		convMap := map[string]interface{}{
			"conv_id":          convID,
			"user_id":          userID,
			"user_nickname":    nicknames[userIdx],
			"cs_id":            csID,
			"source":           sources[rand.Intn(len(sources))],
			"start_time":       startTime,
			"status":           status,
			"create_time":      startTime,
			"update_time":      time.Now(),
			"is_core":          int8(rand.Intn(2)),
			"msg_type":         0,
			"is_manual_adjust": 0,
			"category_id":      0,
			"tags":             "",
		}

		if status == 1 { // ended
			convMap["end_time"] = time.Now()
		} else {
			// For active/transfer, end_time is NULL. We just omit it from map.
		}

		if err := db.Model(&model.Conversation{}).Create(convMap).Error; err == nil {
			seedConvMessage(db, convID, users[userIdx], csID)
		}
	}
	log.Println("Seeded Conversation (Ensured Unique Active Users)")
}

func seedConvMessage(db *gorm.DB, convID, userID, csID string) {
	contents := []string{
		"你好，我想咨询一下产品问题。",
		"请问这个多少钱？",
		"什么时候发货？",
		"质量怎么样？",
		"有优惠吗？",
		"好的，谢谢。",
		"再见。",
		"不太满意。",
		"非常感谢！",
		"稍等一下。",
	}

	count := rand.Intn(10) + 5
	for i := 0; i < count; i++ {
		senderType := int8(rand.Intn(2)) // 0: user, 1: cs
		senderID := userID
		if senderType == 1 {
			senderID = csID
		}

		msg := model.ConvMessage{
			ConvID:       convID,
			SenderType:   senderType,
			SenderID:     senderID,
			MsgContent:   contents[rand.Intn(len(contents))],
			SendTime:     time.Now().Add(-time.Duration(count-i) * time.Minute),
			IsQuickReply: 0,
		}
		db.Create(&msg)
	}
}

func seedLeaveTransfer(db *gorm.DB, csIDs []string) {
	var shifts []model.ShiftConfig
	db.Find(&shifts)
	if len(shifts) == 0 {
		return
	}

	for i := 0; i < 10; i++ {
		csID := csIDs[rand.Intn(len(csIDs))]
		targetDate := time.Now().AddDate(0, 0, rand.Intn(10)+1).Format("2006-01-02")
		applyType := int8(rand.Intn(2)) // 0: leave, 1: transfer

		targetCsID := ""
		if applyType == 1 {
			targetCsID = csIDs[rand.Intn(len(csIDs))]
			if targetCsID == csID {
				continue
			}
		}

		lt := model.LeaveTransfer{
			CsID:           csID,
			ApplyType:      applyType,
			TargetDate:     targetDate,
			ShiftID:        shifts[rand.Intn(len(shifts))].ShiftID,
			TargetCsID:     targetCsID,
			ApprovalStatus: int8(rand.Intn(3)), // 0: pending, 1: approved, 2: rejected
			Reason:         "个人事务",
			CreateTime:     time.Now(),
			UpdateTime:     time.Now(),
		}
		if lt.ApprovalStatus != 0 {
			now := time.Now()
			lt.ApprovalTime = &now
			lt.ApproverID = "ADMIN"
		}
		db.Create(&lt)
	}
	log.Println("Seeded LeaveTransfer")
}
