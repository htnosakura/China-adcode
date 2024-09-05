# 中华人民共和国县以上行政区划代码  

中华人民共和国县以上行政区划代码（2023年更新），来源自[民政部官方数据](https://www.mca.gov.cn/mzsj/xzqh/2023/202301xzqh.html)（感谢https://gist.github.com/mayufo/4207ed3fa925e6b3df7559832af85165?permalink_comment_id=5056538#gistcomment-5056538 提供的信息）

# 数据格式：

|字段|字段说明|
| -----------| ---------------------------------------------------|
|code|行政区划代码（6位数字）|
|name|行政区划名称|
|full_name|行政区划名称（含省、市全称）|
|level|行政区划级别（0代表省级，1代表市级，2代表区县级）|
|province|行政区划所属省名称|
|city|行政区划所属市名称|

# 相关代码（使用Go语言）

通过民政部官方数据excel格式，使用Go语言将信息存储为区域切片，并输出处理后的数据至SQLite数据库。

数据格式：

```undefined
Region struct {
	Code     string
	Name     string
	FullName string
	Level    int // 0: 省, 1: 市, 2: 县
	Province string
	City     string
}
```

‍
