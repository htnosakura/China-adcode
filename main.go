package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time" // 引入 time 包用于简单计时

	_ "github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
)

type Region struct {
	Code     string
	Name     string
	FullName string
	Level    int // 0: 省, 1: 市, 2: 县
	Province string
	City     string
}

func main() {
	startTime := time.Now() // 开始计时

	// 预编译用于检查 6 位数字代码的正则表达式
	var sixDigitCodeRegex = regexp.MustCompile(`^\d{6}$`)

	// --- 1. 打开和读取 Excel 文件 ---
	f, err := excelize.OpenFile("./行政区划.xlsx")
	if err != nil {
		log.Fatalf("无法打开 Excel 文件: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭 Excel 文件失败: %v", err)
		}
	}()

	// 获取指定工作表中的所有行
	rows, err := f.GetRows("Sheet1") // 确保工作表名称正确
	if err != nil {
		log.Fatalf("无法获取工作表 'Sheet1' 的行数据: %v", err)
	}
	log.Printf("读取 Excel 文件 '%s' (工作表 '%s') 成功，共 %d 行。", "./行政区划.xlsx", "Sheet1", len(rows))

	// --- 2. 第一阶段：遍历 Excel，构建基础 Region 列表和省市映射 ---
	log.Println("--- 开始第一阶段：构建基础信息和映射 ---")
	var regions []Region                      // 存储处理后的区划数据（初始只含基础信息）
	var provinceMap = make(map[string]string) // Map: 省代码前缀 (2位) -> 省名
	var cityMap = make(map[string]string)     // Map: 市代码前缀 (4位) -> 市名

	processedCount := 0
	skippedCount := 0
	for i, row := range rows {
		if i == 0 { // 跳过表头行（如果存在）
			continue
		}
		if len(row) < 2 {
			log.Printf("警告：第 %d 行数据列数不足，已跳过。", i+1)
			skippedCount++
			continue // 跳过无效行
		}

		code := strings.TrimSpace(row[0]) // 行政区划代码
		name := strings.TrimSpace(row[1]) // 行政区划名称

		// 检查 code 是否为 6 位数字
		if !sixDigitCodeRegex.MatchString(code) { // <-- 修改在这里
			log.Printf("警告：第 %d 行代码 '%s' 不是一个有效的6位数字代码，已跳过。", i+1, code)
			skippedCount++
			continue
		}

		// 清理名称：去除前后的空格，去除后面的 "*"
		name = strings.TrimSuffix(name, "*")
		name = strings.TrimSpace(name)

		if name == "" {
			log.Printf("警告：第 %d 行名称为空 (Code: %s)，已跳过。", i+1, code)
			skippedCount++
			continue
		}

		region := Region{
			Code: code,
			Name: name,
			// FullName, Province, City 将在第二阶段填充
		}

		// 判断行政区划级别并填充 Map
		if strings.HasSuffix(code, "0000") {
			// 省级
			region.Level = 0
			provinceMap[code[:2]] = name // 记录省级信息
		} else if strings.HasSuffix(code, "00") {
			// 市级
			region.Level = 1
			cityMap[code[:4]] = name // 记录市级信息
		} else {
			// 县级
			region.Level = 2
		}
		regions = append(regions, region) // 添加基础信息到列表
		processedCount++
	}
	log.Printf("--- 第一阶段完成：处理 %d 行，跳过 %d 行。识别省 %d 个，市 %d 个 ---", processedCount, skippedCount, len(provinceMap), len(cityMap))

	// --- 3. 第二阶段：遍历基础 Region 列表，填充详细信息 ---
	log.Println("--- 开始第二阶段：补充省、市及全名信息 ---")
	for i := range regions {
		// 使用指针直接修改 slice 中的元素
		region := &regions[i]
		provinceCodePrefix := region.Code[:2]
		cityCodePrefix := region.Code[:4]

		// 查找并填充省份信息
		provinceName, okProvince := provinceMap[provinceCodePrefix]
		if okProvince {
			region.Province = provinceName
		} else if region.Level != 0 {
			log.Printf("警告：未能找到代码 '%s' (%s) 的省级信息 (前缀 %s)。", region.Code, region.Name, provinceCodePrefix)
		}

		// 查找并填充城市信息（仅对县级）
		if region.Level == 2 {
			cityName, okCity := cityMap[cityCodePrefix]
			if okCity {
				region.City = cityName
			} else {
				// 可能为省直辖县/区，或数据缺失
				log.Printf("未能找到区县代码 '%s' (%s) 的市级信息 (前缀 %s)，可能为省直辖县。", region.Code, region.Name, cityCodePrefix)
			}
		}

		// 构建全名 FullName
		switch region.Level {
		case 0: // 省
			region.FullName = region.Name
		case 1: // 市
			if region.Province != "" {
				region.FullName = region.Province + region.Name
			} else {
				region.FullName = region.Name // 降级处理
			}
		case 2: // 县
			if region.Province != "" && region.City != "" {
				region.FullName = region.Province + region.City + region.Name
			} else if region.Province != "" { // 只有省信息
				region.FullName = region.Province + region.Name
			} else { // 省市信息都无
				region.FullName = region.Name // 最低降级
			}
		}
	}
	log.Println("--- 第二阶段完成 ---")

	// --- 4. 数据库操作 ---
	log.Println("--- 开始数据库操作 ---")
	dbFileName := "行政区划.sqlite"
	db, err := sql.Open("sqlite", dbFileName)
	if err != nil {
		log.Fatalf("无法打开/创建数据库 '%s': %v", dbFileName, err)
	}
	defer db.Close()

	// 创建行政区划表（如果不存在），先删除旧表确保干净运行
	createTableSQL := `
	DROP TABLE IF EXISTS regions;
	CREATE TABLE regions (
		code TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		full_name TEXT,
		level INTEGER NOT NULL,
		province TEXT,
		city TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("创建表 'regions' 失败: %v", err)
	}
	log.Println("数据库表 'regions' 已创建或重置。")

	// 准备插入数据的SQL语句 (使用事务和预编译提高效率)
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("开始数据库事务失败: %v", err)
	}

	insertSQL := `INSERT INTO regions (code, name, full_name, level, province, city) VALUES (?, ?, ?, ?, ?, ?);`
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		_ = tx.Rollback() // 出错时回滚
		log.Fatalf("准备插入数据语句失败: %v", err)
	}
	defer stmt.Close()

	// 将 regions 切片中的数据插入到数据库
	insertCount := 0
	insertErrors := 0
	for _, region := range regions {
		_, err = stmt.Exec(region.Code, region.Name, region.FullName, region.Level, region.Province, region.City)
		if err != nil {
			log.Printf("插入数据失败 (Code: %s): %v", region.Code, err)
			insertErrors++
			// 考虑是否在此处回滚事务，取决于业务需求
		} else {
			insertCount++
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		log.Fatalf("提交数据库事务失败: %v", err)
	}

	log.Printf("--- 数据库操作完成 ---")
	fmt.Printf("数据已成功存储到 SQLite 数据库 '%s'。共插入 %d 条记录。\n", dbFileName, insertCount)
	if insertErrors > 0 {
		fmt.Printf("插入过程中发生 %d 个错误，请检查日志。\n", insertErrors)
	}

	// --- 结束计时 ---
	duration := time.Since(startTime)
	log.Printf("总处理耗时: %s\n", duration)

	// 可选：输出少量处理后的行政区划信息用于验证
	// fmt.Println("\n--- 处理后的部分行政区划信息 ---")
	// limit := 5
	// if len(regions) < limit { limit = len(regions) }
	// for i:=0; i<limit; i++ {
	// 	r := regions[i]
	// 	fmt.Printf("Code: %s, Level: %d, Name: %s, Prov: %s, City: %s, Full: %s\n",
	// 		r.Code, r.Level, r.Name, r.Province, r.City, r.FullName)
	// }
}
