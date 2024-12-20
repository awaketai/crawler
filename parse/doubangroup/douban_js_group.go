package doubangroup

import (
	"github.com/awaketai/crawler/collect"
)

var DouBanGroupJSTask = &collect.TaskMode{
	Propety: collect.Propety{
		Name:     "js_find_douban_sun_room",
		Cookie:   cookie,
		WaitTime: 1,
		MaxDepth: 5,
	},
	Root: rootJs,
	Rules: []collect.RuleMode{
		{
			Name: "解析网站URL",
			ParseFunc: `
				ctx.ParseJSReg("解析阳台房","https://www.douban.com/group/topic/[0-9a-z]+/)\"[^>]*>([^<]+)</a>")";
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

var cookie = `viewed="27043167_25863515_10746113_2243615_36667173_1007305_1091086"; __utma=30149280.1138703939.1688435343.1733140251.1734400228.14; __utma=81379588.927092766.1688435343.1733140311.1734400228.8; _vwo_uuid_v2=D99283ABB7583C723035F3B8EB200F38C|ce7c9a3714600f1a848c69fe82b61426; ll="108288"; bid=p4zwdHrVY7w; __utmz=30149280.1734400228.14.3.utmcsr=help.douban.com|utmccn=(referral)|utmcmd=referral|utmcct=/; _pk_id.100001.3ac3=9e8a4bb305e43b92.1724980792.; __yadk_uid=SryjkZkMMMtjoBtR3bYuWLb6F6XRKX2X; __utmz=81379588.1734400228.8.4.utmcsr=help.douban.com|utmccn=(referral)|utmcmd=referral|utmcct=/; dbcl2="285159894:dMkA02qtf50"; push_noty_num=0; push_doumail_num=0; __utmv=30149280.28515; _vwo_uuid_v2=DA5C0F35C5141ECEE7520D43DF2106264|8d200da2a9f789409ca0ce01e00d2789; _ga_RXNMP372GL=GS1.1.1733140291.1.0.1733140301.50.0.0; _ga=GA1.1.2008343405.1733140291; _pk_ref.100001.3ac3=%5B%22%22%2C%22%22%2C1734400226%2C%22https%3A%2F%2Fhelp.douban.com%2F%22%5D; ck=tQmt; ap_v=0,6.0; _pk_ses.100001.3ac3=1; __utmb=30149280.1.10.1734400228; __utmc=30149280; __utmt_douban=1; __utmb=81379588.1.10.1734400228; __utmc=81379588; __utmt=1`

var rootJs = `
	var arr = new Array();
	for (var i = 0;i <= 50; i+= 25){
		var obj = {
			Url: "https://www.douban.com/group/szsh/discussion?start=" + i,
			Priority: 1,
			RuleName: "解析网站URL",
			Method: "GET"
		}
		arr.push(obj);
		console.log(obj.Url);
	}
	AddJSReqs(arr);
`
