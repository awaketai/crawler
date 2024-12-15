package doubangroup

import (
	"time"

	"github.com/awaketai/crawler/collect"
)

var DouBanGroupJSTask = &collect.TaskMode{
	Propety: collect.Propety{
		Name:     "js_find_douban_sun_room",
		Cookie:   cookie,
		WaitTime: 1 * time.Second,
		MaxDepth: 5,
	},
	Root: rootJs,
	Rules: []collect.RuleMode{
		{
			Name: "解析网站URL",
			ParseFunc: `
				ctx.ParseJSReg("解析阳台房","https://www.douban.com/group/topic/[0-9a-z]+/)\"[^>]*>([^<]+)</a>");
			`,
		},
		{
			Name: "解析阳台房",
			ParseFunc: `
				console.log("parse output");
				ctx.OutputJS("<div class=\"topic-content\">[\\s\\S]*?阳台[\\s\\S]*?<div class=\"aside\">");
			`,
		},
	},
}

var cookie = `viewed="27043167_25863515_10746113_2243615_36667173_1007305_1091086"; __utma=30149280.1138703939.1688435343.1733118222.1733122303.10; ll="108288"; bid=p4zwdHrVY7w; __utmz=30149280.1729597487.8.2.utmcsr=ruanyifeng.com|utmccn=(referral)|utmcmd=referral|utmcct=/; _pk_id.100001.8cb4=18c04f5fb62d2e52.1733118221.; __utmc=30149280; dbcl2="285159894:dMkA02qtf50"; ck=tQmt; push_noty_num=0; push_doumail_num=0; __utmv=30149280.28515; __yadk_uid=3D5K4bndWlX7TLf8CjyAjVV5aB26MFa8; loc-last-index-location-id="108288"; _vwo_uuid_v2=DA5C0F35C5141ECEE7520D43DF2106264|8d200da2a9f789409ca0ce01e00d2789; frodotk_db="4a184671f7672f9cde48d355e6358ed4"; _pk_ses.100001.8cb4=1; __utmb=30149280.26.9.1733123639802; __utmt=1`
var rootJs = `
	var arr = new Array();
	for (var i = 25;i <= 25; i+= 25){
		var obj = {
			Url: "https://www.douban.com/group/szsh/discussion?start=" + i,
			Priority: 1,
			RuleName: "解析网站URL",
			Method: "GET"
		}
		arr.push(obj);
		console.log(arr[0].Url);
		AddJSReq(arr);
	}
`
