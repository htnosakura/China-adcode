package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

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
	f, err := excelize.OpenFile("./行政区划.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// 获取指定工作表中的所有行
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		log.Fatal(err)
	}

	// 存储处理后的区划数据
	var regions []Region
	var provinceMap = make(map[string]string) // 省代码到省名映射
	var cityMap = make(map[string]string)     // 市代码到市名映射

	// 遍历Excel数据
	for _, row := range rows {
		if len(row) < 2 {
			continue // 跳过无效行
		}

		code := row[0] // 行政区划代码
		name := row[1] // 行政区划名称

		// 确保代码长度足够
		if len(code) != 6 {
			log.Printf("跳过无效代码: %s", code)
			continue
		}

		// 去除前后的空格，去除后面的 "*"
		name = strings.TrimSpace(name)
		name = strings.TrimSuffix(name, "*")

		region := Region{
			Code: code,
			Name: name,
		}

		// 判断行政区划级别
		if strings.HasSuffix(code, "0000") {
			// 省级
			region.Level = 0
			region.FullName = name
			provinceMap[code[:2]] = name // 记录省级
		} else if strings.HasSuffix(code, "00") {
			// 市级
			region.Level = 1
			provinceCode := code[:2] + "0000"
			provinceName, ok := provinceMap[provinceCode[:2]]
			if ok {
				region.Province = provinceName
				region.FullName = provinceName + name
				cityMap[code[:4]] = name // 记录市级
			} else {
				region.FullName = name // 如果省级未找到，只使用市名
			}
		} else {
			// 县级
			region.Level = 2
			provinceCode := code[:2] + "0000"
			cityCode := code[:4] + "00"
			provinceName, ok1 := provinceMap[provinceCode[:2]]
			cityName, ok2 := cityMap[cityCode[:4]]
			if ok1 {
				region.Province = provinceName
			}
			if ok2 {
				region.City = cityName
				region.FullName = provinceName + cityName + name // 拼接省+市+县名称
			} else {
				region.FullName = provinceName + name // 如果市不存在，只拼接省+县名称
			}
		}
		regions = append(regions, region)
	}

	// 打开SQLite数据库（如果文件不存在，会自动创建）
	db, err := sql.Open("sqlite", "行政区划.sqlite")
	if err != nil {
		log.Fatalf("无法打开数据库: %v", err)
	}
	defer db.Close()

	// 创建行政区划表（如果不存在）
	createTableSQL := `
CREATE TABLE IF NOT EXISTS regions (
	code TEXT PRIMARY KEY,
	name TEXT,
	full_name TEXT,
	level INTEGER,
	province TEXT,
	city TEXT
);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("创建表失败: %v", err)
	}

	// 准备插入数据的SQL语句
	insertSQL := `INSERT OR REPLACE INTO regions (code, name, full_name, level, province, city) VALUES (?, ?, ?, ?, ?, ?);`
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		log.Fatalf("准备插入数据语句失败: %v", err)
	}
	defer stmt.Close()

	// 将regions切片中的数据插入到数据库
	for _, region := range regions {
		_, err = stmt.Exec(region.Code, region.Name, region.FullName, region.Level, region.Province, region.City)
		if err != nil {
			log.Printf("插入数据失败: %v", err)
		}
	}

	fmt.Println("数据已成功存储到 SQLite 数据库。")

	// 输出处理后的行政区划信息
	for _, region := range regions {
		fmt.Printf("Code: %s, Name: %s, FullName: %s\n", region.Code, region.Name, region.FullName)
	}
}
