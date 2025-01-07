# 中华人民共和国县以上行政区划信息  

中华人民共和国县以上行政区划信息（2023年更新），来源自[民政部官方数据](https://www.mca.gov.cn/mzsj/xzqh/2023/202301xzqh.html)（感谢https://gist.github.com/mayufo/4207ed3fa925e6b3df7559832af85165?permalink_comment_id=5056538#gistcomment-5056538 提供的信息）

### SQLite数据库数据格式

|字段|字段说明|省级示例|市级示例|区县级示例|
| -----------| ---------------------------------------------------| ----------| ----------------| ----------------------|
|code|行政区划代码（6位数字）|130000|130100|130102|
|name|行政区划名称|河北省|石家庄市|长安区|
|full_name|行政区划名称（含省、市全称）|河北省|河北省石家庄市|河北省石家庄市长安区|
|level|行政区划级别（0代表省级，1代表市级，2代表区县级）|0|1|2|
|province|行政区划所属省名称||河北省|河北省|
|city|行政区划所属市名称|||石家庄市|

### 相关格式化代码（使用Go语言）

通过民政部官方数据excel文件，使用Go语言将信息存储为区域切片，并输出处理后的数据至SQLite数据库。

Go行政区划数据格式：

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
